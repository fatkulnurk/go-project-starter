package http

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// statusRecorder captures the response status code for logging. It forwards
// the optional streaming interfaces (Flusher/Hijacker/Pusher) so SSE,
// websockets and server push keep working through the logger.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader records the first status code and forwards it to the wrapped
// writer so downstream middleware can inspect what the handler returned.
func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Flush forwards a flush to the wrapped writer when it supports
// http.Flusher (e.g. for SSE), and is a no-op otherwise.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack forwards the connection to the wrapped writer when it supports
// http.Hijacker; otherwise it returns http.ErrNotSupported.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Push forwards an HTTP/2 server push to the wrapped writer when it supports
// http.Pusher; otherwise it returns http.ErrNotSupported.
func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	if p, ok := r.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

// Unwrap lets http.ResponseController reach the underlying writer.
// It returns the wrapped http.ResponseWriter directly.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// LoggerMiddleware logs every request via slog.
// It records the method, path, status code, duration and request ID (from
// chi's RequestID middleware) as a single "http" log entry.
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}
