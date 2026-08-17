package rbac

import (
	"context"
	"database/sql"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/adapter/api"
	rbaccache "github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/command"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/query"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/infrastructure"
	"github.com/go-chi/chi/v5"
)

// Dependencies are wired by the composition root. Cache and CacheTTL are
// optional; a nil Cache disables the versioned permission cache and a nil
// Auditor skips audit recording.
type Dependencies struct {
	DB       *sql.DB
	DBDriver string
	Cache    cache.Cache
	CacheTTL time.Duration
	Auditor  audit.Recorder
}

// Module wires the RBAC use cases and their adapters. It owns the repositories
// and the permission cache, and exposes them through the API, Service and
// Authorizer surfaces.
type Module struct {
	API         API
	svc         Service
	authz       authorization.Authorizer
	roles       domain.RoleRepository
	permissions domain.PermissionRepository
	bumper      command.CacheBumper
	auditor     audit.Recorder
}

// New constructs the RBAC module. It builds every use case against the given
// dependencies and returns the assembled Module; it never fails, so callers
// can rely on the returned module being fully wired.
func New(deps Dependencies) *Module {
	roles := infrastructure.NewRoleRepository(deps.DB, deps.DBDriver)
	permissions := infrastructure.NewPermissionRepository(deps.DB, deps.DBDriver)
	access := infrastructure.NewUserAccessRepository(deps.DB, deps.DBDriver)

	var pcache *rbaccache.PermissionCache
	if deps.Cache != nil {
		pcache = rbaccache.NewPermissionCache(deps.Cache, deps.CacheTTL)
	}
	// Build the bumper as a plain interface: a typed-nil *PermissionCache
	// wrapped in a CacheBumper interface is NOT nil, which would make the
	// bump* guards below call Bump on a nil receiver. Converting it here
	// keeps a missing cache a real nil so the guards no-op.
	var bumper command.CacheBumper
	if pcache != nil {
		bumper = pcache
	}

	createRole := command.NewCreateRole(roles, deps.Auditor)
	createPermission := command.NewCreatePermission(permissions, deps.Auditor)
	updateRole := command.NewUpdateRole(roles, bumper, deps.Auditor)
	deleteRole := command.NewDeleteRole(roles, bumper, deps.Auditor)
	updatePermission := command.NewUpdatePermission(permissions, bumper, deps.Auditor)
	deletePermission := command.NewDeletePermission(permissions, bumper, deps.Auditor)
	assignRole := command.NewAssignRole(roles, access, bumper, deps.Auditor)
	revokeRole := command.NewRevokeRole(roles, access, bumper, deps.Auditor)
	grantPermission := command.NewGrantPermission(permissions, access, bumper, deps.Auditor)
	revokePermission := command.NewRevokePermission(permissions, access, bumper, deps.Auditor)
	syncRolePermissions := command.NewSyncRolePermissions(roles, permissions, bumper, deps.Auditor)
	getUser := query.NewGetUser(access, pcache)
	getRole := query.NewGetRole(roles)

	svc := &service{getUser: getUser, assignRole: assignRole}

	return &Module{
		API: API{
			CreateRole:          createRole,
			CreatePermission:    createPermission,
			UpdateRole:          updateRole,
			DeleteRole:          deleteRole,
			UpdatePermission:    updatePermission,
			DeletePermission:    deletePermission,
			AssignRole:          assignRole,
			RevokeRole:          revokeRole,
			GrantPermission:     grantPermission,
			RevokePermission:    revokePermission,
			SyncRolePermissions: syncRolePermissions,
			GetUser:             getUser,
			GetRole:             getRole,
			ListRoles:           query.NewListRoles(roles),
			ListPermissions:     query.NewListPermissions(permissions),
		},
		svc:         svc,
		authz:       &Authorizer{svc: svc},
		roles:       roles,
		permissions: permissions,
		bumper:      bumper,
		auditor:     deps.Auditor,
	}
}

// Bootstrap ensures the well-known roles and permissions exist. It is
// idempotent and safe to call on every startup; the composition root should
// run it after the module is built so a fresh database is usable immediately
// (registration relies on the default "user" role existing).
func (m *Module) Bootstrap(ctx context.Context, opts command.BootstrapOptions) error {
	return command.NewBootstrap(m.roles, m.permissions, m.bumper, m.auditor).Execute(ctx, opts)
}

// Service exposes the module's public interface for other modules. It is the
// only sanctioned cross-module dependency (see arch.yaml).
func (m *Module) Service() Service { return m.svc }

// Authorizer exposes the module's authorization implementation for protected
// routes.
func (m *Module) Authorizer() authorization.Authorizer { return m.authz }

// RegisterAPI mounts the module's admin API routes behind auth + rbac.manage.
// It wires every use case from the module's API surface into the chi router.
func (m *Module) RegisterAPI(r chi.Router, authn appauth.Authenticator, authz authorization.Authorizer) {
	api.RegisterRoutes(r, api.Deps{
		CreateRole:          m.API.CreateRole,
		CreatePermission:    m.API.CreatePermission,
		UpdateRole:          m.API.UpdateRole,
		DeleteRole:          m.API.DeleteRole,
		UpdatePermission:    m.API.UpdatePermission,
		DeletePermission:    m.API.DeletePermission,
		AssignRole:          m.API.AssignRole,
		RevokeRole:          m.API.RevokeRole,
		GrantPermission:     m.API.GrantPermission,
		RevokePermission:    m.API.RevokePermission,
		SyncRolePermissions: m.API.SyncRolePermissions,
		GetUser:             m.API.GetUser,
		GetRole:             m.API.GetRole,
		ListRoles:           m.API.ListRoles,
		ListPermissions:     m.API.ListPermissions,
		Authenticator:       authn,
		Authorizer:          authz,
	})
}
