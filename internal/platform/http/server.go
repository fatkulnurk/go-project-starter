// Package http provides the HTTP server and shared middleware for the API.
package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds a router with the standard middleware chain applied.
func NewRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(LoggerMiddleware)
	r.Use(AuditActor)
	r.Use(middleware.Timeout(30 * time.Second))
	return r
}

// NewServer returns an http.Server bound to the router.
func NewServer(port int, router chi.Router) *http.Server {
	return &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
