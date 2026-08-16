package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// DeleteRoleCommand deletes a role by its code.
// Protected roles are refused with ErrProtected; unknown codes are a no-op
// reported as not found.
type DeleteRoleCommand struct {
	Code string
}

// DeleteRole deletes a role and its links. Built-in roles are protected; the
// cache is invalidated after a successful delete.
type DeleteRole struct {
	roles   domain.RoleRepository
	bumper  CacheBumper
	auditor audit.Recorder
}

// NewDeleteRole builds the use case. bumper may be nil to skip cache
// invalidation and auditor may be nil to skip audit recording.
func NewDeleteRole(roles domain.RoleRepository, bumper CacheBumper, auditor audit.Recorder) *DeleteRole {
	return &DeleteRole{roles: roles, bumper: bumper, auditor: auditor}
}

// Execute runs the use case. It returns domain.ErrInvalid on a blank code,
// domain.ErrProtected for built-in roles and domain.ErrNotFound when the role
// does not exist. On success it bumps the cache (checked, one retry) and
// records an audit entry.
func (uc *DeleteRole) Execute(ctx context.Context, cmd DeleteRoleCommand) error {
	code := strings.TrimSpace(cmd.Code)
	if code == "" {
		return domain.ErrInvalid
	}
	if isProtectedRole(code) {
		return domain.ErrProtected
	}
	role, err := uc.roles.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	if role == nil {
		return domain.ErrNotFound
	}
	if err := uc.roles.Delete(ctx, role.ID); err != nil {
		return err
	}
	bumpChecked(ctx, uc.bumper)
	if uc.auditor != nil {
		audit.RecordBestEffort(ctx, uc.auditor, audit.Entry{
			SubjectType: "roles",
			SubjectID:   role.ID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"code": role.Code, "name": role.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
