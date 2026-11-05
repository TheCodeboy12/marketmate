package api

import (
	"context"
	"net/http"
	"strings"

	"rbac-testing/internal/auth"
)

type contextKey string

const userContextKey contextKey = "user_claims"

// ContextWithUser puts the authenticated claims into the request context
func ContextWithUser(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
}

// UserFromContext pulls the authenticated claims back out
func UserFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(userContextKey).(*auth.Claims)
	return claims, ok
}

func (api *API) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get the auth header from the request
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "Missing Authorization Header")
			return
		}

		// extract the token it self
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			writeError(w, http.StatusUnauthorized, "invalid autherazation header format")
			return
		}

		// validating claims
		claims, err := auth.ValidateToken(tokenString, api.jwtSecret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}
		// inject the user into the context of the requests
		ctx := ContextWithUser(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission checks if the authenticated user has the required permission via the Authorizer.
func (api *API) RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			allowed, err := api.authorizer.Can(r.Context(), user, perm)
			if err != nil {
				logger.Printf("authorizer check failed for user %d: %v", user.UserID, err)
				writeError(w, http.StatusInternalServerError, "Internal server error")
				return
			}

			if !allowed {
				writeError(w, http.StatusForbidden, "Forbidden: insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
