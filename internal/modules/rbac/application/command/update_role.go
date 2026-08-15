package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// UpdateRoleCommand renames a role's display label. Code is immutable.
type UpdateRoleCommand struct {
	Code    string
	NewName string
}

// UpdateRole renames a role's display label, keeping its users and permission
// links.
type UpdateRole struct {
	roles   domain.RoleRepository
	bumper  CacheBumper
	auditor audit.Recorder
}

// NewUpdateRole builds the use case.
func NewUpdateRole(roles domain.RoleRepository, bumper CacheBumper, auditor audit.Recorder) *UpdateRole {
	return &UpdateRole{roles: roles, bumper: bumper, auditor: auditor}
}

// Execute runs the use case.
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
		_ = uc.auditor.Record(ctx, audit.Entry{
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
