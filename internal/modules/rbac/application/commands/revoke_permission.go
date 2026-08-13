package commands

import (
	"context"
	"strings"

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
}

// NewRevokePermission builds the use case.
func NewRevokePermission(permissions domain.PermissionRepository, access domain.UserAccessRepository, bumper CacheBumper) *RevokePermission {
	return &RevokePermission{permissions: permissions, access: access, bumper: bumper}
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
	return bump(ctx, uc.bumper)
}
