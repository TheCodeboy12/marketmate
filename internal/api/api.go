// Package api defined the http api
package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

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
	}, api.jwtSecret)
	if err != nil {
		logger.Printf("issue generating jwt token for user %d: %v", u.ID, err)
		writeError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": jwtToken})
}

// pathParam retrieves a path parameter, supporting chi.URLParam as well as http.Request.PathValue.
func pathParam(r *http.Request, key string) string {
	if val := chi.URLParam(r, key); val != "" {
		return val
	}
	return r.PathValue(key)
}

// Routes builds and returns a chi.Router with all configured application endpoints.
func (api *API) Routes() chi.Router {
	r := chi.NewRouter()

	// Public Routes
	r.Post("/login", api.LoginHandler)

	// Protected Routes
	r.Group(func(r chi.Router) {
		r.Use(api.Authenticate)

		r.Get("/me", api.MeHandler)

		// Shopping Lists
		r.Route("/lists", func(r chi.Router) {
			r.With(api.RequirePermission("lists:read")).Get("/", api.ListShoppingLists)
			r.With(api.RequirePermission("lists:create")).Post("/", api.CreateShoppingList)
			r.With(api.RequirePermission("lists:delete")).Delete("/{id}", api.DeleteShoppingList)

			// Items within a list
			r.Route("/{listID}/items", func(r chi.Router) {
				r.With(api.RequirePermission("lists:read")).Get("/", api.ListItemsHandler)
				r.With(api.RequirePermission("items:create")).Post("/", api.CreateItemHandler)
			})
		})

		// Items direct actions
		r.Route("/items", func(r chi.Router) {
			r.With(api.RequirePermission("items:check")).Patch("/{id}/toggle", api.ToggleItemHandler)
			r.With(api.RequirePermission("items:create")).Put("/{id}", api.UpdateListItemHandler)
			r.With(api.RequirePermission("items:delete")).Delete("/{id}", api.DeleteItemHandler)
		})

		// Users
		r.Route("/users", func(r chi.Router) {
			r.With(api.RequirePermission("users:read")).Get("/", api.ListUsersHandler)
			r.With(api.RequirePermission("users:read")).Get("/{userID}", api.GetUserHandler)
			r.With(api.RequirePermission("users:create")).Post("/", api.CreateUserHandler)
			r.With(api.RequirePermission("users:modify")).Put("/{userID}", api.UpdateUserHandler)
			r.With(api.RequirePermission("users:modify")).Put("/{userID}/password", api.UpdateUserPasswordHandler)
			r.With(api.RequirePermission("users:delete")).Delete("/{userID}", api.DeleteUserHandler)
		})
	})

	return r
}
