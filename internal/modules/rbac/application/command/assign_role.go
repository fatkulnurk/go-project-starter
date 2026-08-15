package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// AssignRoleCommand assigns a role (by code) to a user.
type AssignRoleCommand struct {
	UserID string
	Role   string // role code
}

// AssignRole grants a role to a user.
type AssignRole struct {
	roles  domain.RoleRepository
	access domain.UserAccessRepository
	bumper CacheBumper
	audit  audit.Recorder
}

// NewAssignRole builds the use case.
func NewAssignRole(roles domain.RoleRepository, access domain.UserAccessRepository, bumper CacheBumper, auditor audit.Recorder) *AssignRole {
	return &AssignRole{roles: roles, access: access, bumper: bumper, audit: auditor}
}

// Execute runs the use case.
func (uc *AssignRole) Execute(ctx context.Context, cmd AssignRoleCommand) error {
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
	if err := uc.access.AssignRole(ctx, userID, role.ID); err != nil {
		return err
	}
	bumpBestEffort(ctx, uc.bumper)
	if uc.audit != nil {
		_ = uc.audit.Record(ctx, audit.Entry{
			SubjectType: "user_roles",
			SubjectID:   userID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"role_id": role.ID, "role": role.Code},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
