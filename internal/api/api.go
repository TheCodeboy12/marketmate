// Package api defined the http api
package api

import (
	"encoding/json"
	"log"
	"net/http"

	"rbac-testing/internal/auth"
	"rbac-testing/internal/store"
)

var logger = log.Default()

type API struct {
	store     *store.Queries
	jwtSecret []byte
}

// Option represents a functional option for configuring the API instance.
type Option func(*API)

// WithLogger allows overriding the package-level logger.
func WithLogger(l *log.Logger) Option {
	return func(a *API) {
		if l != nil {
			logger = l
		}
	}
}

// SetLogger overrides the package-level logger.
func SetLogger(l *log.Logger) {
	if l != nil {
		logger = l
	}
}

// New creates a new API instance.
func New(store *store.Queries, jwtSecret []byte, opts ...Option) *API {
	api := &API{
		store:     store,
		jwtSecret: jwtSecret,
	}

	for _, opt := range opts {
		opt(api)
	}

	return api
}

type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// writeJSON is a package-level helper that encodes response data as JSON
// and sends the specified HTTP status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Printf("could not encode response: %v", err)
	}
}

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
		logger.Printf("issue generating jwt token for user %d: %v", u.ID, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": jwtToken})
}
