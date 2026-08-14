// Package httpapi is the HTTP adapter of the RBAC module: it parses requests,
// calls one application use case, and renders the standardized response. All
// routes require authentication and the rbac.manage permission.
package api

import (
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/queries"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/go-chi/chi/v5"
)

// Deps bundles the use cases and contracts the handlers need.
type Deps struct {
	CreateRole          *commands.CreateRole
	CreatePermission    *commands.CreatePermission
	AssignRole          *commands.AssignRole
	RevokeRole          *commands.RevokeRole
	GrantPermission     *commands.GrantPermission
	RevokePermission    *commands.RevokePermission
	SyncRolePermissions *commands.SyncRolePermissions
	GetUser             *queries.GetUser
	ListRoles           *queries.ListRoles
	ListPermissions     *queries.ListPermissions
	Authenticator       appauth.Authenticator
	Authorizer          authorization.Authorizer
}

// RegisterRoutes mounts the RBAC admin API under /api/v1/rbac, protected by
// auth + the rbac.manage permission.
func RegisterRoutes(r chi.Router, deps Deps) {
	h := &handler{deps: deps}

	guard := func(group chi.Router) {
		group.Use(platformhttp.Authenticate(deps.Authenticator))
		group.Use(platformhttp.RequirePermission(deps.Authorizer, authorization.PermissionManageRBAC))
	}

	r.Route("/api/v1/rbac", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			guard(r)
			r.Get("/roles", h.listRoles)
			r.Post("/roles", h.createRole)
			r.Get("/permissions", h.listPermissions)
			r.Post("/permissions", h.createPermission)
			r.Put("/roles/{name}/permissions", h.syncRolePermissions)
			r.Get("/users/{userID}", h.getUser)
			r.Post("/users/{userID}/roles", h.assignRole)
			r.Delete("/users/{userID}/roles", h.revokeRole)
			r.Post("/users/{userID}/permissions", h.grantPermission)
			r.Delete("/users/{userID}/permissions", h.revokePermission)
		})
	})
}
