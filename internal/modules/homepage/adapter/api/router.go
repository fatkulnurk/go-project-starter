// Package api is the JSON adapter of the homepage module: it exposes a small
// public endpoint that returns the app branding (name, base URLs) so a
// frontend can render links without hard-coding environment values.
package api

import (
	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage/application"
	"github.com/go-chi/chi/v5"
)

// Deps bundles what the handlers need.
type Deps struct {
	// Info carries the branding exposed by GET /.
	Info application.Info
}

// RegisterRoutes mounts the homepage JSON API at the root.
func RegisterRoutes(r chi.Router, deps Deps) {
	h := &handler{deps: deps}
	r.Get("/", h.info)
}
