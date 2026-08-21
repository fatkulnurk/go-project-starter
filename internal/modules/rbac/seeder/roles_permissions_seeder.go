// Package seeder holds the RBAC module's seeders, mirroring Laravel's
// database/seeders: each seeder is a small struct with a Run method that
// seeds one slice of data. Seeders are wired and run by cmd/seed.
package seeder

import (
	"context"
	"database/sql"

	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/application/seed"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/command"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/infrastructure"
)

// DefaultRoles and DefaultPermissions are the module's built-in seeds: the
// well-known roles and permissions that must always exist. They live here so
// the module's Bootstrap, the seeders and cmd/seed share one definition.
var (
	DefaultRoles = []command.BootstrapRole{
		{Code: authorization.RoleSuperAdmin, Name: "Super Admin"},
		{Code: authorization.RoleUser, Name: "User"},
	}
	DefaultPermissions = []command.BootstrapPermission{
		{Code: authorization.PermissionManageRBAC, Group: "RBAC", Name: "Manage RBAC"},
	}
)

// RolesPermissionsSeeder mirrors Laravel's RoleSeeder/PermissionSeeder: it
// ensures the well-known roles and permissions (plus any extras) exist. It is
// idempotent: existing codes are left untouched.
type RolesPermissionsSeeder struct {
	db               *sql.DB
	dbDriver         string
	extraRoles       []command.BootstrapRole
	extraPermissions []command.BootstrapPermission
}

// NewRolesPermissionsSeeder builds the seeder. Extras may be nil; when given,
// they are ensured in addition to the module's built-in DefaultRoles and
// DefaultPermissions.
func NewRolesPermissionsSeeder(db *sql.DB, dbDriver string, extraRoles []command.BootstrapRole, extraPermissions []command.BootstrapPermission) *RolesPermissionsSeeder {
	return &RolesPermissionsSeeder{db: db, dbDriver: dbDriver, extraRoles: extraRoles, extraPermissions: extraPermissions}
}

// Register wires the roles/permissions seeder into a seed registry under
// "rbac.roles_permissions".
func Register(reg *seed.Registry, db *sql.DB, dbDriver string) {
	reg.Register("rbac.roles_permissions", NewRolesPermissionsSeeder(db, dbDriver, nil, nil))
}

// Run implements seed.Seed. It bootstraps the built-in plus any extra roles
// and permissions through the module's Bootstrap use case; the operation is
// idempotent (existing codes are left untouched) and returns an error when a
// repository write fails.
func (s *RolesPermissionsSeeder) Run(ctx context.Context) error {
	roles := infrastructure.NewRoleRepository(s.db, s.db, s.dbDriver)
	permissions := infrastructure.NewPermissionRepository(s.db, s.db, s.dbDriver)
	bootstrap := command.NewBootstrap(roles, permissions, nil, nil)
	return bootstrap.Execute(ctx, command.BootstrapOptions{
		DefaultRoles:       append(append([]command.BootstrapRole{}, DefaultRoles...), s.extraRoles...),
		DefaultPermissions: append(append([]command.BootstrapPermission{}, DefaultPermissions...), s.extraPermissions...),
	})
}

var _ seed.Seed = (*RolesPermissionsSeeder)(nil)
