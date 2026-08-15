package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// DeleteRoleCommand deletes a role by its code.
type DeleteRoleCommand struct {
	Code string
}

// DeleteRole deletes a role and its links.
type DeleteRole struct {
	roles   domain.RoleRepository
	bumper  CacheBumper
	auditor audit.Recorder
}

// NewDeleteRole builds the use case.
func NewDeleteRole(roles domain.RoleRepository, bumper CacheBumper, auditor audit.Recorder) *DeleteRole {
	return &DeleteRole{roles: roles, bumper: bumper, auditor: auditor}
}

// Execute runs the use case.
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
	bumpBestEffort(ctx, uc.bumper)
	if uc.auditor != nil {
		_ = uc.auditor.Record(ctx, audit.Entry{
			SubjectType: "roles",
			SubjectID:   role.ID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"code": role.Code, "name": role.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
