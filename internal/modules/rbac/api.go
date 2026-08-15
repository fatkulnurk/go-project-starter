package rbac

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/command"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/query"
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
	return s.assignRole.Execute(ctx, command.AssignRoleCommand{UserID: userID, Role: roleName})
}
