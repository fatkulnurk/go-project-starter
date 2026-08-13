package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// CreatePermissionCommand creates a named permission.
type CreatePermissionCommand struct {
	Name string
}

// CreatePermission persists a new permission.
type CreatePermission struct {
	permissions domain.PermissionRepository
}

// NewCreatePermission builds the use case.
func NewCreatePermission(permissions domain.PermissionRepository) *CreatePermission {
	return &CreatePermission{permissions: permissions}
}

// Execute runs the use case.
func (uc *CreatePermission) Execute(ctx context.Context, cmd CreatePermissionCommand) error {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return domain.ErrInvalid
	}
	existing, err := uc.permissions.FindByName(ctx, name)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.ErrConflict
	}
	perm, err := domain.NewPermission(name)
	if err != nil {
		return err
	}
	return uc.permissions.Save(ctx, perm)
}
