package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// DeletePermissionCommand deletes a permission by its code.
type DeletePermissionCommand struct {
	Code string
}

// DeletePermission deletes a permission and its links.
type DeletePermission struct {
	permissions domain.PermissionRepository
	bumper      CacheBumper
	auditor     audit.Auditor
}

// NewDeletePermission builds the use case.
func NewDeletePermission(permissions domain.PermissionRepository, bumper CacheBumper, auditor audit.Auditor) *DeletePermission {
	return &DeletePermission{permissions: permissions, bumper: bumper, auditor: auditor}
}

// Execute runs the use case.
func (uc *DeletePermission) Execute(ctx context.Context, cmd DeletePermissionCommand) error {
	code := strings.TrimSpace(cmd.Code)
	if code == "" {
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
	if err := uc.permissions.Delete(ctx, perm.ID); err != nil {
		return err
	}
	bumpBestEffort(ctx, uc.bumper)
	if uc.auditor != nil {
		_ = uc.auditor.Record(ctx, audit.Entry{
			SubjectType: "permissions",
			SubjectID:   perm.ID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"code": perm.Code, "group": perm.Group, "name": perm.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
