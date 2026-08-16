package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// GrantPermissionCommand grants a permission (by code) directly to a user.
// Direct grants apply regardless of the user's roles.
type GrantPermissionCommand struct {
	UserID     string
	Permission string // permission code
}

// GrantPermission grants a direct user permission. The cache is invalidated
// after a successful grant so the change takes effect quickly.
type GrantPermission struct {
	permissions domain.PermissionRepository
	access      domain.UserAccessRepository
	bumper      CacheBumper
	audit       audit.Recorder
}

// NewGrantPermission builds the use case. bumper may be nil to skip cache
// invalidation and auditor may be nil to skip audit recording.
func NewGrantPermission(permissions domain.PermissionRepository, access domain.UserAccessRepository, bumper CacheBumper, auditor audit.Recorder) *GrantPermission {
	return &GrantPermission{permissions: permissions, access: access, bumper: bumper, audit: auditor}
}

// Execute runs the use case. It returns domain.ErrInvalid on blank user/permission
// and domain.ErrNotFound when the permission does not exist. On success it
// bumps the cache (checked, one retry) and records an audit entry.
func (uc *GrantPermission) Execute(ctx context.Context, cmd GrantPermissionCommand) error {
	userID := strings.TrimSpace(cmd.UserID)
	code := strings.TrimSpace(cmd.Permission)
	if userID == "" || code == "" {
		return domain.ErrInvalid
	}
	perm, err := uc.permissions.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	if perm == nil {
		return domain.ErrNotFound
	}
	if err := uc.access.GrantPermission(ctx, userID, perm.ID); err != nil {
		return err
	}
	bumpChecked(ctx, uc.bumper)
	if uc.audit != nil {
		audit.RecordBestEffort(ctx, uc.audit, audit.Entry{
			SubjectType: "user_permissions",
			SubjectID:   userID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"permission_id": perm.ID, "permission": perm.Code},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
