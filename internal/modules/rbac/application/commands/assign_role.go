package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// AssignRoleCommand assigns a role (by name) to a user.
type AssignRoleCommand struct {
	UserID string
	Role   string
}

// AssignRole grants a role to a user.
type AssignRole struct {
	roles  domain.RoleRepository
	access domain.UserAccessRepository
	bumper CacheBumper
	audit  audit.Auditor
}

// NewAssignRole builds the use case.
func NewAssignRole(roles domain.RoleRepository, access domain.UserAccessRepository, bumper CacheBumper, auditor audit.Auditor) *AssignRole {
	return &AssignRole{roles: roles, access: access, bumper: bumper, audit: auditor}
}

// Execute runs the use case.
func (uc *AssignRole) Execute(ctx context.Context, cmd AssignRoleCommand) error {
	userID := strings.TrimSpace(cmd.UserID)
	roleName := strings.TrimSpace(cmd.Role)
	if userID == "" || roleName == "" {
		return domain.ErrInvalid
	}
	role, err := uc.roles.FindByName(ctx, roleName)
	if err != nil {
		return err
	}
	if role == nil {
		return domain.ErrNotFound
	}
	if err := uc.access.AssignRole(ctx, userID, role.ID); err != nil {
		return err
	}
	if err := bump(ctx, uc.bumper); err != nil {
		return err
	}
	if uc.audit != nil {
		_ = uc.audit.Record(ctx, audit.Entry{
			SubjectType: "user_roles",
			SubjectID:   userID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"role_id": role.ID, "role": role.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
