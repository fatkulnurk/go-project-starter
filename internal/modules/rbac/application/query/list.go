package query

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// ListRoles returns all roles with their permission codes, ordered by creation
// time.
type ListRoles struct {
	roles domain.RoleRepository
}

// NewListRoles builds the use case. It depends only on the RoleRepository and
// fetches permission codes in one query to avoid the N+1 pattern.
func NewListRoles(roles domain.RoleRepository) *ListRoles {
	return &ListRoles{roles: roles}
}

// Execute runs the use case. It returns every role with its permission codes,
// ordered by creation time; the permission set for all roles is resolved in a
// single query to avoid the N+1 pattern. Errors from the repository are
// propagated unchanged.
func (q *ListRoles) Execute(ctx context.Context) ([]RoleDetail, error) {
	roles, err := q.roles.List(ctx)
	if err != nil {
		return nil, err
	}
	permsByRole, err := q.roles.PermissionsForAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RoleDetail, 0, len(roles))
	for _, r := range roles {
		out = append(out, RoleDetail{ID: r.ID, Code: r.Code, Name: r.Name, Permissions: permsByRole[r.ID]})
	}
	return out, nil
}

// ListPermissions returns all permissions ordered by name, as full domain
// permission records for the admin UI.
type ListPermissions struct {
	permissions domain.PermissionRepository
}

// NewListPermissions builds the use case. It wraps a PermissionRepository and
// returns every permission via the repository's ordered List.
func NewListPermissions(permissions domain.PermissionRepository) *ListPermissions {
	return &ListPermissions{permissions: permissions}
}

// Execute runs the use case. It returns every permission ordered by group then
// name, propagating repository errors unchanged.
func (q *ListPermissions) Execute(ctx context.Context) ([]*domain.Permission, error) {
	return q.permissions.List(ctx)
}
