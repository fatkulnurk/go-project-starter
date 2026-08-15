package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// SyncRolePermissionsCommand replaces the permission set of a role. Missing
// permissions are created on the fly (their display name and group default to
// the code and its prefix), mirroring Spatie's syncPermissions.
type SyncRolePermissionsCommand struct {
	Role        string   // role code
	Permissions []string // permission codes
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
	role, err := uc.roles.FindByCode(ctx, strings.TrimSpace(cmd.Role))
	if err != nil {
		return err
	}
	if role == nil {
		return domain.ErrNotFound
	}

	oldCodes, err := uc.roles.PermissionsFor(ctx, role.ID)
	if err != nil {
		return err
	}
	var ids []string
	var newCodes []string
	for _, raw := range cmd.Permissions {
		code := strings.TrimSpace(raw)
		if code == "" {
			continue
		}
		perm, err := uc.permissions.FindByCode(ctx, code)
		if err != nil {
			return err
		}
		if perm == nil {
			perm, err = domain.NewPermission(code, groupFor(code), code)
			if err != nil {
				return err
			}
			if err := uc.permissions.Save(ctx, perm); err != nil {
				return err
			}
		}
		ids = append(ids, perm.ID)
		newCodes = append(newCodes, perm.Code)
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
			OldValues:   map[string]any{"role": role.Code, "permissions": oldCodes},
			NewValues:   map[string]any{"role": role.Code, "permissions": newCodes},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}

// groupFor derives a display group from a permission code prefix, falling back
// to "General" for codes without a dot.
func groupFor(code string) string {
	if i := strings.IndexByte(code, '.'); i > 0 {
		return code[:i]
	}
	return "General"
}
