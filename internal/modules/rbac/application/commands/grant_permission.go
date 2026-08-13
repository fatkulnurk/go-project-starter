package commands

import (
	"context"
	"strings"

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
}

// NewGrantPermission builds the use case.
func NewGrantPermission(permissions domain.PermissionRepository, access domain.UserAccessRepository, bumper CacheBumper) *GrantPermission {
	return &GrantPermission{permissions: permissions, access: access, bumper: bumper}
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
	return bump(ctx, uc.bumper)
}
