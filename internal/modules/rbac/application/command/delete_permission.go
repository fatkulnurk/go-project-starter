package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// DeletePermissionCommand deletes a permission by its code.
// Protected permissions are refused with ErrProtected; unknown codes are a
// no-op reported as not found.
type DeletePermissionCommand struct {
	Code string
}

// DeletePermission deletes a permission and its links. Built-in permissions
// are protected; the cache is invalidated after a successful delete.
type DeletePermission struct {
	permissions domain.PermissionRepository
	bumper      CacheBumper
	auditor     audit.Recorder
}

// NewDeletePermission builds the use case. bumper may be nil to skip cache
// invalidation and auditor may be nil to skip audit recording.
func NewDeletePermission(permissions domain.PermissionRepository, bumper CacheBumper, auditor audit.Recorder) *DeletePermission {
	return &DeletePermission{permissions: permissions, bumper: bumper, auditor: auditor}
}

// Execute runs the use case. It returns domain.ErrInvalid on a blank code,
// domain.ErrProtected for built-in permissions and domain.ErrNotFound when the
// permission does not exist. On success it bumps the cache (checked, one
// retry) and records an audit entry.
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
	bumpChecked(ctx, uc.bumper)
	if uc.auditor != nil {
		audit.RecordBestEffort(ctx, uc.auditor, audit.Entry{
			SubjectType: "permissions",
			SubjectID:   perm.ID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"code": perm.Code, "group": perm.Group, "name": perm.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
