package http

import (
	"net/http"
	"strings"
)

// SecurityHeaders sets safe-by-default response headers on every request.
// Options supplied via the functional option pattern; see the constants below
// for sane defaults.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Content-Security-Policy", "default-src 'self'")
			next.ServeHTTP(w, r)
		})
	}
}

// appendVary adds value to the Vary header without clobbering values other
// middleware already set, so caches never serve the wrong variant.
func appendVary(h http.Header, value string) {
	existing := h.Get("Vary")
	if existing == "" {
		h.Set("Vary", value)
		return
	}
	if !strings.Contains(existing, value) {
		h.Set("Vary", existing+", "+value)
	}
}

// CORS lets browsers call the API from the given origins. An empty list
// disables cross-origin access (same-origin only). Preflight requests are
// answered directly.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allow := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allow[o] = true
	}
	allowedMethods := "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	allowedHeaders := "Authorization, Content-Type"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(allow) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			origin := r.Header.Get("Origin")
			h := w.Header()
			if origin != "" && allow[origin] {
				h.Set("Access-Control-Allow-Origin", origin)
				appendVary(h, "Origin")
				h.Set("Access-Control-Allow-Credentials", "true")
			}
			if r.Method == http.MethodOptions {
				if origin != "" && allow[origin] {
					h.Set("Access-Control-Allow-Methods", allowedMethods)
					h.Set("Access-Control-Allow-Headers", allowedHeaders)
					h.Set("Access-Control-Max-Age", "86400")
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
