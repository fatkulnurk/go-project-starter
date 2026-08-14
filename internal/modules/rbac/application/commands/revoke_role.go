package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// RevokeRoleCommand removes a role (by name) from a user.
type RevokeRoleCommand struct {
	UserID string
	Role   string
}

// RevokeRole removes a role from a user.
type RevokeRole struct {
	roles  domain.RoleRepository
	access domain.UserAccessRepository
	bumper CacheBumper
	audit  audit.Auditor
}

// NewRevokeRole builds the use case.
func NewRevokeRole(roles domain.RoleRepository, access domain.UserAccessRepository, bumper CacheBumper, auditor audit.Auditor) *RevokeRole {
	return &RevokeRole{roles: roles, access: access, bumper: bumper, audit: auditor}
}

// Execute runs the use case.
func (uc *RevokeRole) Execute(ctx context.Context, cmd RevokeRoleCommand) error {
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
	if err := uc.access.RevokeRole(ctx, userID, role.ID); err != nil {
		return err
	}
	if err := bump(ctx, uc.bumper); err != nil {
		return err
	}
	if uc.audit != nil {
		_ = uc.audit.Record(ctx, audit.Entry{
			SubjectType: "user_roles",
			SubjectID:   userID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"role_id": role.ID, "role": role.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
