package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// UpdatePermissionCommand updates a permission's display group and label. Code
// is immutable.
type UpdatePermissionCommand struct {
	Code     string
	NewGroup string
	NewName  string
}

// UpdatePermission updates a permission's display group and label, keeping its
// code, role and user links.
type UpdatePermission struct {
	permissions domain.PermissionRepository
	bumper      CacheBumper
	auditor     audit.Auditor
}

// NewUpdatePermission builds the use case.
func NewUpdatePermission(permissions domain.PermissionRepository, bumper CacheBumper, auditor audit.Auditor) *UpdatePermission {
	return &UpdatePermission{permissions: permissions, bumper: bumper, auditor: auditor}
}

// Execute runs the use case.
func (uc *UpdatePermission) Execute(ctx context.Context, cmd UpdatePermissionCommand) error {
	code := strings.TrimSpace(cmd.Code)
	newGroup := strings.TrimSpace(cmd.NewGroup)
	newName := strings.TrimSpace(cmd.NewName)
	if code == "" || newName == "" {
		return domain.ErrInvalid
	}
	if isProtectedPermission(code) {
		return domain.ErrProtected
	}
	perm, err := uc.permissions.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	if perm == nil {
		return domain.ErrNotFound
	}
	if newGroup == "" {
		newGroup = perm.Group
	}
	if newGroup == perm.Group && newName == perm.Name {
		return nil
	}
	if err := uc.permissions.Update(ctx, perm.ID, newGroup, newName); err != nil {
		return err
	}
	bumpBestEffort(ctx, uc.bumper)
	if uc.auditor != nil {
		_ = uc.auditor.Record(ctx, audit.Entry{
			SubjectType: "permissions",
			SubjectID:   perm.ID,
			Action:      audit.ActionUpdated,
			OldValues:   map[string]any{"code": perm.Code, "group": perm.Group, "name": perm.Name},
			NewValues:   map[string]any{"code": perm.Code, "group": newGroup, "name": newName},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
