package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// CreatePermissionCommand creates a permission from a stable code, a display
// group and a display name.
type CreatePermissionCommand struct {
	Code  string
	Group string
	Name  string
}

// CreatePermission persists a new permission.
type CreatePermission struct {
	permissions domain.PermissionRepository
	auditor     audit.Recorder
}

// NewCreatePermission builds the use case.
func NewCreatePermission(permissions domain.PermissionRepository, auditor audit.Recorder) *CreatePermission {
	return &CreatePermission{permissions: permissions, auditor: auditor}
}

// Execute runs the use case.
func (uc *CreatePermission) Execute(ctx context.Context, cmd CreatePermissionCommand) error {
	code := strings.TrimSpace(cmd.Code)
	name := strings.TrimSpace(cmd.Name)
	if code == "" || name == "" {
		return domain.ErrInvalid
	}
	existing, err := uc.permissions.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.ErrConflict
	}
	perm, err := domain.NewPermission(code, strings.TrimSpace(cmd.Group), name)
	if err != nil {
		return err
	}
	if err := uc.permissions.Save(ctx, perm); err != nil {
		return err
	}
	if uc.auditor != nil {
		_ = uc.auditor.Record(ctx, audit.Entry{
			SubjectType: "permissions",
			SubjectID:   perm.ID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"code": perm.Code, "group": perm.Group, "name": perm.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
