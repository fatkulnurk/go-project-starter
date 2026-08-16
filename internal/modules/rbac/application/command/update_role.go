package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// UpdateRoleCommand renames a role's display label. Code is immutable.
// The role's users and permission links are preserved.
type UpdateRoleCommand struct {
	Code    string
	NewName string
}

// UpdateRole renames a role's display label, keeping its users and permission
// links. Built-in roles are protected; the cache is invalidated on change.
type UpdateRole struct {
	roles   domain.RoleRepository
	bumper  CacheBumper
	auditor audit.Recorder
}

// NewUpdateRole builds the use case. bumper may be nil to skip cache
// invalidation and auditor may be nil to skip audit recording.
func NewUpdateRole(roles domain.RoleRepository, bumper CacheBumper, auditor audit.Recorder) *UpdateRole {
	return &UpdateRole{roles: roles, bumper: bumper, auditor: auditor}
}

// Execute runs the use case. It returns domain.ErrInvalid on blank input,
// domain.ErrProtected for built-in roles and domain.ErrNotFound when the role
// does not exist. A rename to the current name is a successful no-op; a real
// change bumps the cache best-effort and records an audit entry.
func (uc *UpdateRole) Execute(ctx context.Context, cmd UpdateRoleCommand) error {
	code := strings.TrimSpace(cmd.Code)
	newName := strings.TrimSpace(cmd.NewName)
	if code == "" || newName == "" {
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
	if newName == role.Name {
		return nil
	}
	if err := uc.roles.UpdateName(ctx, role.ID, newName); err != nil {
		return err
	}
	bumpBestEffort(ctx, uc.bumper)
	if uc.auditor != nil {
		audit.RecordBestEffort(ctx, uc.auditor, audit.Entry{
			SubjectType: "roles",
			SubjectID:   role.ID,
			Action:      audit.ActionUpdated,
			OldValues:   map[string]any{"code": role.Code, "name": role.Name},
			NewValues:   map[string]any{"code": role.Code, "name": newName},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
