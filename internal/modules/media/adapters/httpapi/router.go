// Package httpapi is the HTTP adapter of the media module: it parses uploads,
// calls one application use case, and renders the standardized response.
package httpapi

import (
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/application/queries"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/go-chi/chi/v5"
)

// Deps bundles the use cases and contracts the handlers need.
type Deps struct {
	AddMedia      *commands.AddMedia
	RemoveMedia   *commands.RemoveMedia
	GetMedia      *queries.GetMedia
	ListByModel   *queries.ListByModel
	Authenticator appauth.Authenticator
	Authorizer    authorization.Authorizer
	// MaxUploadSize caps the request body for uploads, in bytes.
	MaxUploadSize int64
}

// RegisterRoutes mounts the media API under /api/v1/media. Reads require
// authentication; writes additionally require the media.manage permission.
func RegisterRoutes(r chi.Router, deps Deps) {
	h := &handler{deps: deps}

	r.Route(mediaBasePath, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(platformhttp.Authenticate(deps.Authenticator))
			r.Get("/", h.list)
			r.Get("/{id}", h.get)
			r.Get("/{id}/download", h.download)
		})
		r.Group(func(r chi.Router) {
			r.Use(platformhttp.Authenticate(deps.Authenticator))
			r.Use(platformhttp.RequirePermission(deps.Authorizer, authorization.PermissionManageMedia))
			r.Post("/", h.upload)
			r.Delete("/{id}", h.remove)
		})
	})
}
