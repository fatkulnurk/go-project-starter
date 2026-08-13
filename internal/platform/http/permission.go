package http

import (
	"net/http"

	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
)

// RequirePermission guards a route with an authorization check. The request
// must already carry an authenticated identity (place it after
// Authenticate); a missing identity returns 401, a denied action returns 403.
func RequirePermission(authz authorization.Authorizer, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := appauth.IdentityFrom(r.Context())
			if id == nil {
				WriteError(w, http.StatusUnauthorized, "unauthenticated", "unauthenticated")
				return
			}
			if err := authz.Can(r.Context(), authorization.Identity{UserID: id.UserID, Roles: id.Roles}, action, nil); err != nil {
				WriteMappedError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole guards a route with a role check. The request must already
// carry an authenticated identity (place it after Authenticate); a missing
// identity returns 401, a denied role returns 403.
func RequireRole(authz authorization.Authorizer, role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := appauth.IdentityFrom(r.Context())
			if id == nil {
				WriteError(w, http.StatusUnauthorized, "unauthenticated", "unauthenticated")
				return
			}
			if err := authz.CanRole(r.Context(), authorization.Identity{UserID: id.UserID, Roles: id.Roles}, role); err != nil {
				WriteMappedError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
