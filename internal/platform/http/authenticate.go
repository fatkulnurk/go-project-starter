package http

import (
	"net/http"
	"strings"

	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
)

// Authenticate is middleware that resolves a Bearer token into an Identity
// via the application Authenticator contract and stores it in the context.
// Unauthenticated requests get a 401 JSON envelope.
func Authenticate(authenticator appauth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := BearerToken(r)
			if raw == "" {
				writeUnauthenticated(w)
				return
			}
			id, err := authenticator.Authenticate(r.Context(), raw)
			if err != nil {
				writeUnauthenticated(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(appauth.WithIdentity(r.Context(), id)))
		})
	}
}

// BearerToken extracts the token from the Authorization header.
func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func writeUnauthenticated(w http.ResponseWriter) {
	WriteError(w, http.StatusUnauthorized, "unauthenticated", "unauthenticated")
}
