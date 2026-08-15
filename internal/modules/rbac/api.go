package rbac

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/queries"
)

// Service is the public surface other modules may depend on. It resolves
// roles and permissions (by code) for a user and grants roles (by code).
type Service interface {
	RolesAndPermissions(ctx context.Context, userID string) (roles, permissions []string, err error)
	HasPermission(ctx context.Context, userID, permission string) (bool, error)
	HasRole(ctx context.Context, userID, role string) (bool, error)
	AssignRole(ctx context.Context, userID, roleName string) error
}

// API exposes the module's use cases.
type API struct {
	CreateRole          *commands.CreateRole
	CreatePermission    *commands.CreatePermission
	UpdateRole          *commands.UpdateRole
	DeleteRole          *commands.DeleteRole
	UpdatePermission    *commands.UpdatePermission
	DeletePermission    *commands.DeletePermission
	AssignRole          *commands.AssignRole
	RevokeRole          *commands.RevokeRole
	GrantPermission     *commands.GrantPermission
	RevokePermission    *commands.RevokePermission
	SyncRolePermissions *commands.SyncRolePermissions
	GetUser             *queries.GetUser
	GetRole             *queries.GetRole
	ListRoles           *queries.ListRoles
	ListPermissions     *queries.ListPermissions
}

type service struct {
	getUser    *queries.GetUser
	assignRole *commands.AssignRole
}

func (s *service) RolesAndPermissions(ctx context.Context, userID string) ([]string, []string, error) {
	res, err := s.getUser.Execute(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	return res.Roles, res.Permissions, nil
}

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

func (s *service) AssignRole(ctx context.Context, userID, roleName string) error {
	return s.assignRole.Execute(ctx, commands.AssignRoleCommand{UserID: userID, Role: roleName})
}
