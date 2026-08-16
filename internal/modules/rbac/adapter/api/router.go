// Package httpapi is the HTTP adapter of the RBAC module: it parses requests,
// calls one application use case, and renders the standardized response. All
// routes require authentication and the rbac.manage permission.
package api

import (
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/command"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/query"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/go-chi/chi/v5"
)

// Deps bundles the use cases and contracts the handlers need. All use-case
// pointers must be non-nil; Authenticator and Authorizer gate every route.
type Deps struct {
	CreateRole          *command.CreateRole
	CreatePermission    *command.CreatePermission
	UpdateRole          *command.UpdateRole
	DeleteRole          *command.DeleteRole
	UpdatePermission    *command.UpdatePermission
	DeletePermission    *command.DeletePermission
	AssignRole          *command.AssignRole
	RevokeRole          *command.RevokeRole
	GrantPermission     *command.GrantPermission
	RevokePermission    *command.RevokePermission
	SyncRolePermissions *command.SyncRolePermissions
	GetUser             *query.GetUser
	GetRole             *query.GetRole
	ListRoles           *query.ListRoles
	ListPermissions     *query.ListPermissions
	Authenticator       appauth.Authenticator
	Authorizer          authorization.Authorizer
}

// RegisterRoutes mounts the RBAC admin API under /api/v1/rbac, protected by
// auth + the rbac.manage permission.
func RegisterRoutes(r chi.Router, deps Deps) {
	h := &handler{deps: deps}

	guard := func(group chi.Router) {
		group.Use(platformhttp.AuditActor)
		group.Use(platformhttp.Authenticate(deps.Authenticator))
		group.Use(platformhttp.RequirePermission(deps.Authorizer, authorization.PermissionManageRBAC))
	}

	r.Route("/api/v1/rbac", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			guard(r)
			r.Get("/roles", h.listRoles)
			r.Post("/roles", h.createRole)
			r.Get("/roles/{code}", h.getRole)
			r.Put("/roles/{code}", h.updateRole)
			r.Delete("/roles/{code}", h.deleteRole)
			r.Put("/roles/{code}/permissions", h.syncRolePermissions)
			r.Get("/permissions", h.listPermissions)
			r.Post("/permissions", h.createPermission)
			r.Put("/permissions/{code}", h.updatePermission)
			r.Delete("/permissions/{code}", h.deletePermission)
			r.Get("/users/{userID}", h.getUser)
			r.Post("/users/{userID}/roles", h.assignRole)
			r.Delete("/users/{userID}/roles", h.revokeRole)
			r.Post("/users/{userID}/permissions", h.grantPermission)
			r.Delete("/users/{userID}/permissions", h.revokePermission)
		})
	})
}
