package http

import (
	"net"
	"net/http"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
)

// AuditActor is middleware that captures the request's IP address and user
// agent and stores an audit.Actor in the context. For protected routes the
// Authenticate middleware runs afterwards and upgrades the actor to the
// authenticated user; otherwise the actor stays a "system" caller.
func AuditActor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor := audit.Actor{
			Type:      audit.ActorSystem,
			IPAddress: ClientIP(r),
			UserAgent: r.UserAgent(),
		}
		if id := appauth.IdentityFrom(r.Context()); id != nil {
			actor.Type = audit.ActorUser
			actor.ID = id.UserID
		}
		next.ServeHTTP(w, r.WithContext(audit.WithActor(r.Context(), actor)))
	})
}

// ClientIP returns the caller's IP address, honouring X-Forwarded-For.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if parts := strings.Split(xff, ","); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
