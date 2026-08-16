package command

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

// SyncRolePermissions replaces a role's permissions. Missing permissions are
// created on the fly and the cache is invalidated after the replacement.
type SyncRolePermissions struct {
	roles       domain.RoleRepository
	permissions domain.PermissionRepository
	bumper      CacheBumper
	audit       audit.Recorder
}

// NewSyncRolePermissions builds the use case. bumper may be nil to skip cache
// invalidation and auditor may be nil to skip audit recording.
func NewSyncRolePermissions(roles domain.RoleRepository, permissions domain.PermissionRepository, bumper CacheBumper, auditor audit.Recorder) *SyncRolePermissions {
	return &SyncRolePermissions{roles: roles, permissions: permissions, bumper: bumper, audit: auditor}
}

// Execute runs the use case. It returns domain.ErrNotFound when the role does
// not exist, deduplicates repeated permission codes and creates missing ones
// (group derived from the code prefix), then atomically replaces the role's
// permission set, bumps the cache (checked, one retry) and records an audit
// entry. Repository errors are propagated unchanged.
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
	seen := make(map[string]struct{})
	for _, raw := range cmd.Permissions {
		code := strings.TrimSpace(raw)
		if code == "" {
			continue
		}
		if _, dup := seen[code]; dup {
			// Deduplicate: a repeated code would violate the composite PK.
			continue
		}
		seen[code] = struct{}{}
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
	bumpChecked(ctx, uc.bumper)
	if uc.audit != nil {
		audit.RecordBestEffort(ctx, uc.audit, audit.Entry{
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
