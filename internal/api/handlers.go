package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"rbac-testing/internal/auth"
	"rbac-testing/internal/store"
)

func (api *API) MeHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	me, err := api.store.GetUserById(r.Context(), user.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Unable to find you")
		return
	}

	writeJSON(w, http.StatusOK, me)
}

// GetUserHandler fetches a single user by ID
func (api *API) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	uID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil || uID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid user ID provided")
		return
	}

	u, err := api.store.GetUserById(r.Context(), uID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Unable to find user %s", userID))
		return
	}

	writeJSON(w, http.StatusOK, u)
}

// ListUsersHandler lists users with an optional limit query parameter
func (api *API) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	limit := int64(20) // default limit when ?limit is omitted
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		l, err := strconv.ParseUint(limitStr, 10, 64)
		if err != nil || l == 0 {
			writeError(w, http.StatusBadRequest, "Invalid limit parameter")
			return
		}
		limit = int64(l)
	}

	users, err := api.store.ListUsers(r.Context(), limit)
	if err != nil {
		logger.Printf("failed to fetch users: %v", err)
		writeError(w, http.StatusInternalServerError, "Unable to fetch users")
		return
	}

	writeJSON(w, http.StatusOK, users)
}

type CreateUserRequest struct {
	TenantID int64  `json:"tenant_id"`
	RoleID   int64  `json:"role_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CreateUserHandler handles creating a new user
func (api *API) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Fallback to caller's tenant_id if not explicitly provided
	if req.TenantID <= 0 {
		if caller, ok := UserFromContext(r.Context()); ok {
			req.TenantID = caller.TenantID
		}
	}

	if req.TenantID <= 0 {
		writeError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}
	if req.RoleID <= 0 {
		writeError(w, http.StatusBadRequest, "role_id is required")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		logger.Printf("failed to hash password: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to process password")
		return
	}

	newUser, err := api.store.CreateUser(r.Context(), store.CreateUserParams{
		TenantID:     req.TenantID,
		RoleID:       req.RoleID,
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: sql.NullString{String: hashedPassword, Valid: true},
	})
	if err != nil {
		logger.Printf("failed to create user: %v", err)
		writeError(w, http.StatusBadRequest, "Unable to create user (email may already exist)")
		return
	}

	writeJSON(w, http.StatusCreated, newUser)
}

type UpdateUserRequest struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	RoleID   *int64  `json:"role_id"`
	Password *string `json:"password"`
}

// UpdateUserHandler handles updating an existing user's details and/or password
func (api *API) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	uID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil || uID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid user ID provided")
		return
	}

	existing, err := api.store.GetUserById(r.Context(), uID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Unable to find user %s", userID))
		return
	}

	if caller, ok := UserFromContext(r.Context()); ok {
		if caller.TenantID != existing.TenantID {
			writeError(w, http.StatusForbidden, "Forbidden: cannot modify users in another tenant")
			return
		}
	}

	var req UpdateUserRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	name := existing.Name
	if req.Name != nil && *req.Name != "" {
		name = *req.Name
	}
	email := existing.Email
	if req.Email != nil && *req.Email != "" {
		email = *req.Email
	}
	roleID := existing.RoleID
	if req.RoleID != nil && *req.RoleID > 0 {
		roleID = *req.RoleID
	}

	updated, err := api.store.UpdateUser(r.Context(), store.UpdateUserParams{
		ID:     uID,
		Name:   name,
		Email:  email,
		RoleID: roleID,
	})
	if err != nil {
		logger.Printf("failed to update user %d: %v", uID, err)
		writeError(w, http.StatusBadRequest, "Unable to update user")
		return
	}

	if req.Password != nil && *req.Password != "" {
		hashed, err := auth.HashPassword(*req.Password)
		if err != nil {
			logger.Printf("failed to hash password for user %d: %v", uID, err)
			writeError(w, http.StatusInternalServerError, "Failed to process password")
			return
		}

		if err := api.store.UpdateUserPassword(r.Context(), store.UpdateUserPasswordParams{
			ID:           uID,
			PasswordHash: sql.NullString{String: hashed, Valid: true},
		}); err != nil {
			logger.Printf("failed to update password for user %d: %v", uID, err)
			writeError(w, http.StatusInternalServerError, "Failed to update password")
			return
		}
	}

	writeJSON(w, http.StatusOK, updated)
}

// DeleteUserHandler handles deleting an existing user
func (api *API) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	uID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil || uID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid user ID provided")
		return
	}

	existing, err := api.store.GetUserById(r.Context(), uID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Unable to find user %s", userID))
		return
	}

	if caller, ok := UserFromContext(r.Context()); ok {
		if caller.TenantID != existing.TenantID {
			writeError(w, http.StatusForbidden, "Forbidden: cannot delete users from another tenant")
			return
		}
		if caller.UserID == uID {
			writeError(w, http.StatusBadRequest, "Cannot delete your own account")
			return
		}
	}

	if err := api.store.DeleteUser(r.Context(), uID); err != nil {
		logger.Printf("failed to delete user %d: %v", uID, err)
		writeError(w, http.StatusInternalServerError, "Unable to delete user")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}
