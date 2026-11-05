// Package api defined the http api
package api

import (
	"encoding/json"
	"log"
	"net/http"

	"rbac-testing/internal/auth"
	"rbac-testing/internal/store"
)

type API struct {
	store     *store.Queries
	jwtSecret []byte
}

func New(store *store.Queries, jwtSecret []byte) *API {
	return &API{
		store:     store,
		jwtSecret: jwtSecret,
	}
}

type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("could not encode response: %v", err)
	}
}

// LoginHandler handles /login
func (api *API) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var data LoginPayload
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid request body",
		})
		return
	}

	// 1. Fetch user by email using request context
	u, err := api.store.GetUserByEmail(r.Context(), data.Email)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Invalid email or password",
		})
		return
	}

	// 2. Compare stored password hash against incoming plaintext password
	if !auth.CheckPassword(u.PasswordHash.String, data.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"message": "Invalid email or password",
		})
		return
	}

	// 3. Issue JWT token
	jwtToken, err := auth.GenerateToken(auth.Claims{
		UserID:   u.ID,
		TenantID: u.TenantID,
		RoleID:   u.RoleID,
	}, api.jwtSecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Internal server error",
		})
		log.Printf("issue generating jwt token for user %d: %v", u.ID, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": jwtToken})
}
