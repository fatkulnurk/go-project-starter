package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// RevokePermissionCommand revokes a direct user permission (by name).
type RevokePermissionCommand struct {
	UserID     string
	Permission string
}

// RevokePermission revokes a direct user permission.
type RevokePermission struct {
	permissions domain.PermissionRepository
	access      domain.UserAccessRepository
	bumper      CacheBumper
	audit       audit.Auditor
}

// NewRevokePermission builds the use case.
func NewRevokePermission(permissions domain.PermissionRepository, access domain.UserAccessRepository, bumper CacheBumper, auditor audit.Auditor) *RevokePermission {
	return &RevokePermission{permissions: permissions, access: access, bumper: bumper, audit: auditor}
}

// Execute runs the use case.
func (uc *RevokePermission) Execute(ctx context.Context, cmd RevokePermissionCommand) error {
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
	if err := uc.access.RevokePermission(ctx, userID, perm.ID); err != nil {
		return err
	}
	bumpBestEffort(ctx, uc.bumper)
	if uc.audit != nil {
		_ = uc.audit.Record(ctx, audit.Entry{
			SubjectType: "user_permissions",
			SubjectID:   userID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"permission_id": perm.ID, "permission": perm.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
