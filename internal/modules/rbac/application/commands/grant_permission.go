package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// GrantPermissionCommand grants a permission (by name) directly to a user.
type GrantPermissionCommand struct {
	UserID     string
	Permission string
}

// GrantPermission grants a direct user permission.
type GrantPermission struct {
	permissions domain.PermissionRepository
	access      domain.UserAccessRepository
	bumper      CacheBumper
	audit       audit.Auditor
}

// NewGrantPermission builds the use case.
func NewGrantPermission(permissions domain.PermissionRepository, access domain.UserAccessRepository, bumper CacheBumper, auditor audit.Auditor) *GrantPermission {
	return &GrantPermission{permissions: permissions, access: access, bumper: bumper, audit: auditor}
}

// Execute runs the use case.
func (uc *GrantPermission) Execute(ctx context.Context, cmd GrantPermissionCommand) error {
	userID := strings.TrimSpace(cmd.UserID)
	name := strings.TrimSpace(cmd.Permission)
	if userID == "" || name == "" {
		return domain.ErrInvalid
	}
	perm, err := uc.permissions.FindByName(ctx, name)
	if err != nil {
		return err
	}
	if perm == nil {
		return domain.ErrNotFound
	}
	if err := uc.access.GrantPermission(ctx, userID, perm.ID); err != nil {
		return err
	}
	bumpBestEffort(ctx, uc.bumper)
	if uc.audit != nil {
		_ = uc.audit.Record(ctx, audit.Entry{
			SubjectType: "user_permissions",
			SubjectID:   userID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"permission_id": perm.ID, "permission": perm.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
