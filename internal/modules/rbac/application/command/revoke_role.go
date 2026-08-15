package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// RevokeRoleCommand removes a role (by code) from a user.
type RevokeRoleCommand struct {
	UserID string
	Role   string // role code
}

// RevokeRole removes a role from a user.
type RevokeRole struct {
	roles  domain.RoleRepository
	access domain.UserAccessRepository
	bumper CacheBumper
	audit  audit.Recorder
}

// NewRevokeRole builds the use case.
func NewRevokeRole(roles domain.RoleRepository, access domain.UserAccessRepository, bumper CacheBumper, auditor audit.Recorder) *RevokeRole {
	return &RevokeRole{roles: roles, access: access, bumper: bumper, audit: auditor}
}

// Execute runs the use case.
func (uc *RevokeRole) Execute(ctx context.Context, cmd RevokeRoleCommand) error {
	userID := strings.TrimSpace(cmd.UserID)
	roleCode := strings.TrimSpace(cmd.Role)
	if userID == "" || roleCode == "" {
		return domain.ErrInvalid
	}
	role, err := uc.roles.FindByCode(ctx, roleCode)
	if err != nil {
		return err
	}
	if role == nil {
		return domain.ErrNotFound
	}
	if err := uc.access.RevokeRole(ctx, userID, role.ID); err != nil {
		return err
	}
	bumpBestEffort(ctx, uc.bumper)
	if uc.audit != nil {
		_ = uc.audit.Record(ctx, audit.Entry{
			SubjectType: "user_roles",
			SubjectID:   userID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"role_id": role.ID, "role": role.Code},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
