package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// SyncRolePermissionsCommand replaces the permission set of a role. Missing
// permissions are created on the fly, mirroring Spatie's syncPermissions.
type SyncRolePermissionsCommand struct {
	Role        string
	Permissions []string
}

// SyncRolePermissions replaces a role's permissions.
type SyncRolePermissions struct {
	roles       domain.RoleRepository
	permissions domain.PermissionRepository
	bumper      CacheBumper
}

// NewSyncRolePermissions builds the use case.
func NewSyncRolePermissions(roles domain.RoleRepository, permissions domain.PermissionRepository, bumper CacheBumper) *SyncRolePermissions {
	return &SyncRolePermissions{roles: roles, permissions: permissions, bumper: bumper}
}

// Execute runs the use case.
func (uc *SyncRolePermissions) Execute(ctx context.Context, cmd SyncRolePermissionsCommand) error {
	role, err := uc.roles.FindByName(ctx, strings.TrimSpace(cmd.Role))
	if err != nil {
		return err
	}
	if role == nil {
		return domain.ErrNotFound
	}

	var ids []string
	for _, raw := range cmd.Permissions {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		perm, err := uc.permissions.FindByName(ctx, name)
		if err != nil {
			return err
		}
		if perm == nil {
			perm, err = domain.NewPermission(name)
			if err != nil {
				return err
			}
			if err := uc.permissions.Save(ctx, perm); err != nil {
				return err
			}
		}
		ids = append(ids, perm.ID)
	}
	if err := uc.roles.SetPermissions(ctx, role.ID, ids); err != nil {
		return err
	}
	return bump(ctx, uc.bumper)
}
