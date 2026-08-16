package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// BootstrapRole is a well-known role to ensure on startup. Code is the stable
// machine identifier and Name the display label used when the role is created.
type BootstrapRole struct {
	Code string
	Name string
}

// BootstrapPermission is a well-known permission to ensure on startup. Group
// and Name are display metadata used when the permission is created.
type BootstrapPermission struct {
	Code  string
	Group string
	Name  string
}

// BootstrapOptions lists the roles and permissions to ensure on startup. Every
// entry is trimmed of surrounding whitespace; blank codes are skipped.
type BootstrapOptions struct {
	DefaultRoles       []BootstrapRole
	DefaultPermissions []BootstrapPermission
}

// Bootstrap seeds the well-known roles and permissions so admin APIs can rely
// on them existing. It is idempotent and safe to call on every start.
type Bootstrap struct {
	roles       domain.RoleRepository
	permissions domain.PermissionRepository
	bumper      CacheBumper
	audit       audit.Recorder
}

// NewBootstrap builds the use case. bumper may be nil to skip cache
// invalidation and auditor may be nil to skip audit recording.
func NewBootstrap(roles domain.RoleRepository, permissions domain.PermissionRepository, bumper CacheBumper, auditor audit.Recorder) *Bootstrap {
	return &Bootstrap{roles: roles, permissions: permissions, bumper: bumper, audit: auditor}
}

// Execute runs the use case. It creates each missing permission and role,
// leaving existing codes untouched, then invalidates the cache best-effort.
// It returns the first repository error encountered; a nil auditor and nil
// bumper are both no-ops.
func (uc *Bootstrap) Execute(ctx context.Context, opts BootstrapOptions) error {
	for _, p := range opts.DefaultPermissions {
		if err := uc.ensurePermission(ctx, strings.TrimSpace(p.Code), strings.TrimSpace(p.Group), strings.TrimSpace(p.Name)); err != nil {
			return err
		}
	}
	for _, r := range opts.DefaultRoles {
		if err := uc.ensureRole(ctx, strings.TrimSpace(r.Code), strings.TrimSpace(r.Name)); err != nil {
			return err
		}
	}
	bumpBestEffort(ctx, uc.bumper)
	return nil
}

func (uc *Bootstrap) ensurePermission(ctx context.Context, code, group, name string) error {
	if code == "" {
		return nil
	}
	existing, err := uc.permissions.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	perm, err := domain.NewPermission(code, group, name)
	if err != nil {
		return err
	}
	if err := uc.permissions.Save(ctx, perm); err != nil {
		return err
	}
	if uc.audit != nil {
		audit.RecordBestEffort(ctx, uc.audit, audit.Entry{
			SubjectType: "permissions",
			SubjectID:   perm.ID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"code": perm.Code, "group": perm.Group, "name": perm.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}

func (uc *Bootstrap) ensureRole(ctx context.Context, code, name string) error {
	if code == "" {
		return nil
	}
	existing, err := uc.roles.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	role, err := domain.NewRole(code, name)
	if err != nil {
		return err
	}
	if err := uc.roles.Save(ctx, role); err != nil {
		return err
	}
	if uc.audit != nil {
		audit.RecordBestEffort(ctx, uc.audit, audit.Entry{
			SubjectType: "roles",
			SubjectID:   role.ID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"code": role.Code, "name": role.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
