package rbac

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/command"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/query"
)

// Service is the public surface other modules may depend on. It resolves
// roles and permissions (by code) for a user and grants roles (by code).
// Implementations must treat empty role/permission codes as trivially allowed
// when used through the authorizer.
type Service interface {
	// RolesAndPermissions returns the user's role codes and effective
	// permission codes (direct grants plus role inheritance), or an error
	// when the lookup fails.
	RolesAndPermissions(ctx context.Context, userID string) (roles, permissions []string, err error)
	// HasPermission reports whether the user holds the permission. Super
	// admins always hold every permission.
	HasPermission(ctx context.Context, userID, permission string) (bool, error)
	// HasRole reports whether the user holds the role (matched by code).
	// Lookup errors are propagated unchanged.
	HasRole(ctx context.Context, userID, role string) (bool, error)
	// AssignRole grants the role (by code) to the user, returning
	// domain.ErrNotFound when the role does not exist and domain.ErrProtected
	// for privilege-escalating super_admin grants.
	AssignRole(ctx context.Context, userID, roleName string) error
}

// API exposes the module's use cases to the HTTP adapter and the composition
// root. Each field is one ready-to-execute command or query.
type API struct {
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
}

type service struct {
	getUser    *query.GetUser
	assignRole *command.AssignRole
}

// RolesAndPermissions implements Service. It delegates to the GetUser query
// and returns its role codes and effective permission codes, propagating any
// lookup error.
func (s *service) RolesAndPermissions(ctx context.Context, userID string) ([]string, []string, error) {
	res, err := s.getUser.Execute(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	return res.Roles, res.Permissions, nil
}

// HasPermission implements Service. It returns true when the user holds the
// permission directly, inherits it through a role, or holds the super_admin
// role (which bypasses all checks). Lookup errors are propagated unchanged.
func (s *service) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	res, err := s.getUser.Execute(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, r := range res.Roles {
		if r == authorization.RoleSuperAdmin {
			return true, nil
		}
	}
	for _, p := range res.Permissions {
		if p == permission {
			return true, nil
		}
	}
	return false, nil
}

// HasRole implements Service. It returns true when the user holds the role
// (matched by code) and propagates any lookup error.
func (s *service) HasRole(ctx context.Context, userID, role string) (bool, error) {
	res, err := s.getUser.Execute(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, r := range res.Roles {
		if r == role {
			return true, nil
		}
	}
	return false, nil
}

// AssignRole implements Service. It delegates to the AssignRole command with
// the role resolved by code and propagates its error contract (domain.ErrInvalid
// on blank input, domain.ErrNotFound on unknown role, domain.ErrProtected on
// privilege-escalating super_admin grants).
func (s *service) AssignRole(ctx context.Context, userID, roleName string) error {
	return s.assignRole.Execute(ctx, command.AssignRoleCommand{UserID: userID, Role: roleName})
}
