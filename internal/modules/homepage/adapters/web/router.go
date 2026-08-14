// Package web is the web adapter of the homepage module: it renders the
// public landing page at / from the platform base view layout. No business
// logic or SQL lives here.
package web

import (
	"github.com/fatkulnurk/go-project-starter/internal/platform/view"
	"github.com/go-chi/chi/v5"
)

// Deps bundles what the handlers need.
type Deps struct {
	// Common carries branding for the rendered page.
	Common view.Common
}

// RegisterRoutes mounts the homepage routes.
func RegisterRoutes(r chi.Router, deps Deps) {
	h := &handler{deps: deps}
	r.Get("/", h.welcome)
}
