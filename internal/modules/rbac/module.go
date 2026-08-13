// Package rbac is the role-based access control module: roles, permissions,
// role↔user and permission↔user links, plus the authorizer used by protected
// endpoints. Other modules depend only on rbac.Service (see api.go).
package rbac

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/apierr"
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/adapters/httpapi"
	rbaccache "github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/queries"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/infrastructure"
	"github.com/go-chi/chi/v5"
)

// Default roles and permissions seeded on startup.
var (
	defaultRoles       = []string{authorization.RoleSuperAdmin, authorization.RoleUser}
	defaultPermissions = []string{authorization.PermissionManageRBAC, authorization.PermissionManageMedia}
)

// Dependencies are wired by the composition root.
type Dependencies struct {
	DB       *sql.DB
	DBDriver string
	Cache    cache.Cache
	CacheTTL time.Duration
}

// BootstrapOptions drives startup seeding and super admin promotion.
type BootstrapOptions struct {
	SuperAdminEmail string
	// FindUserID resolves an email to a user ID; used to promote the super
	// admin. Provided by the composition root via the auth module.
	FindUserID func(ctx context.Context, email string) (string, error)
}

// Module wires the RBAC use cases and their adapters.
type Module struct {
	API   API
	svc   Service
	authz authorization.Authorizer
}

// New constructs the RBAC module.
func New(deps Dependencies) *Module {
	roles := infrastructure.NewRoleRepository(deps.DB, deps.DBDriver)
	permissions := infrastructure.NewPermissionRepository(deps.DB, deps.DBDriver)
	access := infrastructure.NewUserAccessRepository(deps.DB, deps.DBDriver)

	var pcache *rbaccache.PermissionCache
	if deps.Cache != nil {
		pcache = rbaccache.NewPermissionCache(deps.Cache, deps.CacheTTL)
	}

	createRole := commands.NewCreateRole(roles)
	createPermission := commands.NewCreatePermission(permissions)
	assignRole := commands.NewAssignRole(roles, access, pcache)
	revokeRole := commands.NewRevokeRole(roles, access, pcache)
	grantPermission := commands.NewGrantPermission(permissions, access, pcache)
	revokePermission := commands.NewRevokePermission(permissions, access, pcache)
	syncRolePermissions := commands.NewSyncRolePermissions(roles, permissions, pcache)
	bootstrap := commands.NewBootstrap(roles, permissions, pcache)
	getUser := queries.NewGetUser(access, pcache)

	svc := &service{getUser: getUser, assignRole: assignRole}

	return &Module{
		API: API{
			CreateRole:          createRole,
			CreatePermission:    createPermission,
			AssignRole:          assignRole,
			RevokeRole:          revokeRole,
			GrantPermission:     grantPermission,
			RevokePermission:    revokePermission,
			SyncRolePermissions: syncRolePermissions,
			Bootstrap:           bootstrap,
			GetUser:             getUser,
			ListRoles:           queries.NewListRoles(roles),
			ListPermissions:     queries.NewListPermissions(permissions),
		},
		svc:   svc,
		authz: &Authorizer{svc: svc},
	}
}

// Service exposes the module's public interface for other modules.
func (m *Module) Service() Service { return m.svc }

// Authorizer exposes the module's authorization implementation for protected
// routes.
func (m *Module) Authorizer() authorization.Authorizer { return m.authz }

// Bootstrap seeds the well-known roles and permissions, then promotes the
// configured super admin email (skipped silently when the user does not exist
// yet).
func (m *Module) Bootstrap(ctx context.Context, opts BootstrapOptions) error {
	if err := m.API.Bootstrap.Execute(ctx, commands.BootstrapOptions{
		DefaultRoles:       defaultRoles,
		DefaultPermissions: defaultPermissions,
	}); err != nil {
		return err
	}
	if opts.SuperAdminEmail == "" || opts.FindUserID == nil {
		return nil
	}
	userID, err := opts.FindUserID(ctx, opts.SuperAdminEmail)
	if err != nil {
		if errors.Is(err, apierr.ErrNotFound) {
			return nil
		}
		return err
	}
	return m.svc.AssignRole(ctx, userID, authorization.RoleSuperAdmin)
}

// RegisterHTTP mounts the module's admin routes behind auth + rbac.manage.
func (m *Module) RegisterHTTP(r chi.Router, authn appauth.Authenticator, authz authorization.Authorizer) {
	httpapi.RegisterRoutes(r, httpapi.Deps{
		CreateRole:          m.API.CreateRole,
		CreatePermission:    m.API.CreatePermission,
		AssignRole:          m.API.AssignRole,
		RevokeRole:          m.API.RevokeRole,
		GrantPermission:     m.API.GrantPermission,
		RevokePermission:    m.API.RevokePermission,
		SyncRolePermissions: m.API.SyncRolePermissions,
		GetUser:             m.API.GetUser,
		ListRoles:           m.API.ListRoles,
		ListPermissions:     m.API.ListPermissions,
		Authenticator:       authn,
		Authorizer:          authz,
	})
}
