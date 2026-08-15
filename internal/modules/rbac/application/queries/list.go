package queries

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// ListRoles returns all roles with their permission codes, ordered by creation
// time.
type ListRoles struct {
	roles domain.RoleRepository
}

// NewListRoles builds the use case.
func NewListRoles(roles domain.RoleRepository) *ListRoles {
	return &ListRoles{roles: roles}
}

// Execute runs the use case.
func (q *ListRoles) Execute(ctx context.Context) ([]RoleDetail, error) {
	roles, err := q.roles.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RoleDetail, 0, len(roles))
	for _, r := range roles {
		perms, err := q.roles.PermissionsFor(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, RoleDetail{ID: r.ID, Code: r.Code, Name: r.Name, Permissions: perms})
	}
	return out, nil
}

// ListPermissions returns all permissions ordered by name.
type ListPermissions struct {
	permissions domain.PermissionRepository
}

// NewListPermissions builds the use case.
func NewListPermissions(permissions domain.PermissionRepository) *ListPermissions {
	return &ListPermissions{permissions: permissions}
}

// Execute runs the use case.
func (q *ListPermissions) Execute(ctx context.Context) ([]*domain.Permission, error) {
	return q.permissions.List(ctx)
}
