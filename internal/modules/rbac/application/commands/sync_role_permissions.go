package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
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
	audit       audit.Auditor
}

// NewSyncRolePermissions builds the use case.
func NewSyncRolePermissions(roles domain.RoleRepository, permissions domain.PermissionRepository, bumper CacheBumper, auditor audit.Auditor) *SyncRolePermissions {
	return &SyncRolePermissions{roles: roles, permissions: permissions, bumper: bumper, audit: auditor}
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

	oldNames, err := uc.roles.PermissionsFor(ctx, role.ID)
	if err != nil {
		return err
	}
	var ids []string
	var newNames []string
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
		newNames = append(newNames, perm.Name)
	}
	if err := uc.roles.SetPermissions(ctx, role.ID, ids); err != nil {
		return err
	}
	bumpBestEffort(ctx, uc.bumper)
	if uc.audit != nil {
		_ = uc.audit.Record(ctx, audit.Entry{
			SubjectType: "role_permissions",
			SubjectID:   role.ID,
			Action:      audit.ActionUpdated,
			OldValues:   map[string]any{"role": role.Name, "permissions": oldNames},
			NewValues:   map[string]any{"role": role.Name, "permissions": newNames},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
