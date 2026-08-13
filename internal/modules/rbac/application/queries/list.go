package queries

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// ListRoles returns all roles ordered by creation time.
type ListRoles struct {
	roles domain.RoleRepository
}

// NewListRoles builds the use case.
func NewListRoles(roles domain.RoleRepository) *ListRoles {
	return &ListRoles{roles: roles}
}

// Execute runs the use case.
func (q *ListRoles) Execute(ctx context.Context) ([]*domain.Role, error) {
	return q.roles.List(ctx)
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
