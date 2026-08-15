package query

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// RoleDetail is a role with its permission codes.
type RoleDetail struct {
	ID          string
	Code        string
	Name        string
	Permissions []string
}

// GetRole resolves a role by code with its permission set.
type GetRole struct {
	roles domain.RoleRepository
}

// NewGetRole builds the use case.
func NewGetRole(roles domain.RoleRepository) *GetRole {
	return &GetRole{roles: roles}
}

// Execute runs the use case.
func (q *GetRole) Execute(ctx context.Context, code string) (*RoleDetail, error) {
	role, err := q.roles.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, domain.ErrNotFound
	}
	perms, err := q.roles.PermissionsFor(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	return &RoleDetail{ID: role.ID, Code: role.Code, Name: role.Name, Permissions: perms}, nil
}
