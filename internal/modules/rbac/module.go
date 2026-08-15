package rbac

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/apierr"
	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/adapters/api"
	rbaccache "github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/queries"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/infrastructure"
	"github.com/go-chi/chi/v5"
)

// Default roles and permissions seeded on startup. Code is the stable
// identifier; Name (and Group for permissions) are display labels shown in the
// admin UI.
var (
	defaultRoles = []commands.BootstrapRole{
		{Code: authorization.RoleSuperAdmin, Name: "Super Admin"},
		{Code: authorization.RoleUser, Name: "User"},
	}
	defaultPermissions = []commands.BootstrapPermission{
		{Code: authorization.PermissionManageRBAC, Group: "RBAC", Name: "Manage RBAC"},
	}
)

// Dependencies are wired by the composition root.
type Dependencies struct {
	DB       *sql.DB
	DBDriver string
	Cache    cache.Cache
	CacheTTL time.Duration
	Auditor  audit.Auditor
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

	createRole := commands.NewCreateRole(roles, deps.Auditor)
	createPermission := commands.NewCreatePermission(permissions, deps.Auditor)
	updateRole := commands.NewUpdateRole(roles, pcache, deps.Auditor)
	deleteRole := commands.NewDeleteRole(roles, pcache, deps.Auditor)
	updatePermission := commands.NewUpdatePermission(permissions, pcache, deps.Auditor)
	deletePermission := commands.NewDeletePermission(permissions, pcache, deps.Auditor)
	assignRole := commands.NewAssignRole(roles, access, pcache, deps.Auditor)
	revokeRole := commands.NewRevokeRole(roles, access, pcache, deps.Auditor)
	grantPermission := commands.NewGrantPermission(permissions, access, pcache, deps.Auditor)
	revokePermission := commands.NewRevokePermission(permissions, access, pcache, deps.Auditor)
	syncRolePermissions := commands.NewSyncRolePermissions(roles, permissions, pcache, deps.Auditor)
	bootstrap := commands.NewBootstrap(roles, permissions, pcache, deps.Auditor)
	getUser := queries.NewGetUser(access, pcache)
	getRole := queries.NewGetRole(roles)

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
			Bootstrap:           bootstrap,
			GetUser:             getUser,
			GetRole:             getRole,
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

// RegisterAPI mounts the module's admin API routes behind auth + rbac.manage.
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
