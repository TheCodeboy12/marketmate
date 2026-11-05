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
	store      store.Querier
	jwtSecret  []byte
	authorizer auth.Authorizer
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

// New creates a new API instance with decoupled store and authorizer.
func New(store store.Querier, jwtSecret []byte, authorizer auth.Authorizer, opts ...Option) *API {
	api := &API{
		store:      store,
		jwtSecret:  jwtSecret,
		authorizer: authorizer,
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

// readJSON decodes the request body into dst and enforces a 1MB payload limit.
func readJSON(r *http.Request, dst any) error {
	maxBytes := int64(1 << 20) // 1 MB limit
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)

	dec := json.NewDecoder(r.Body)
	return dec.Decode(dst)
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

// writeError writes a standardized JSON error message response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}

func (api *API) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var data LoginPayload
	if err := readJSON(r, &data); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 1. Fetch user by email and verify password
	u, err := api.store.GetUserByEmail(r.Context(), data.Email)
	if err != nil || !auth.CheckPassword(u.PasswordHash.String, data.Password) {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// 2. Issue JWT token
	jwtToken, err := auth.GenerateToken(auth.Claims{
		UserID:   u.ID,
		TenantID: u.TenantID,
		RoleID:   u.RoleID,
	}, api.jwtSecret)
	if err != nil {
		logger.Printf("issue generating jwt token for user %d: %v", u.ID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": jwtToken})
}
