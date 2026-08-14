package http

import (
	"net"
	"net/http"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
)

// trustedProxies is the set of IPs/CIDRs whose X-Forwarded-For header the
// server may trust. Set once via SetTrustedProxies; empty means the header is
// always ignored.
var trustedProxies []*net.IPNet

// SetTrustedProxies configures which proxies may set X-Forwarded-For. Entries
// may be individual IPs or CIDR blocks.
func SetTrustedProxies(entries []string) {
	trustedProxies = nil
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if strings.Contains(e, "/") {
			if _, ipnet, err := net.ParseCIDR(e); err == nil {
				trustedProxies = append(trustedProxies, ipnet)
			}
			continue
		}
		if ip := net.ParseIP(e); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			trustedProxies = append(trustedProxies, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		}
	}
}

// isTrusted reports whether remote is a configured trusted proxy.
func isTrusted(remote string) bool {
	if len(trustedProxies) == 0 {
		return false
	}
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range trustedProxies {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIP returns the caller's real IP address. When the direct peer is a
// configured trusted proxy, the right-most untrusted X-Forwarded-For entry is
// used; otherwise X-Forwarded-For is ignored entirely to prevent spoofing.
func ClientIP(r *http.Request) string {
	if isTrusted(r.RemoteAddr) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			for i := len(parts) - 1; i >= 0; i-- {
				ip := strings.TrimSpace(parts[i])
				if ip != "" && !isTrusted(ip) {
					return ip
				}
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

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
