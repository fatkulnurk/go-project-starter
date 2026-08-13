package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// BootstrapOptions lists the roles and permissions to ensure on startup.
type BootstrapOptions struct {
	DefaultRoles       []string
	DefaultPermissions []string
}

// Bootstrap seeds the well-known roles and permissions so admin APIs can rely
// on them existing. It is idempotent and safe to call on every start.
type Bootstrap struct {
	roles       domain.RoleRepository
	permissions domain.PermissionRepository
	bumper      CacheBumper
}

// NewBootstrap builds the use case.
func NewBootstrap(roles domain.RoleRepository, permissions domain.PermissionRepository, bumper CacheBumper) *Bootstrap {
	return &Bootstrap{roles: roles, permissions: permissions, bumper: bumper}
}

// Execute runs the use case.
func (uc *Bootstrap) Execute(ctx context.Context, opts BootstrapOptions) error {
	for _, name := range opts.DefaultPermissions {
		if err := uc.ensurePermission(ctx, strings.TrimSpace(name)); err != nil {
			return err
		}
	}
	for _, name := range opts.DefaultRoles {
		if err := uc.ensureRole(ctx, strings.TrimSpace(name)); err != nil {
			return err
		}
	}
	return bump(ctx, uc.bumper)
}

func (uc *Bootstrap) ensurePermission(ctx context.Context, name string) error {
	if name == "" {
		return nil
	}
	existing, err := uc.permissions.FindByName(ctx, name)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	perm, err := domain.NewPermission(name)
	if err != nil {
		return err
	}
	return uc.permissions.Save(ctx, perm)
}

func (uc *Bootstrap) ensureRole(ctx context.Context, name string) error {
	if name == "" {
		return nil
	}
	existing, err := uc.roles.FindByName(ctx, name)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	role, err := domain.NewRole(name)
	if err != nil {
		return err
	}
	return uc.roles.Save(ctx, role)
}
