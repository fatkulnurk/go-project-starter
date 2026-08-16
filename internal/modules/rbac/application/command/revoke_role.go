package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// RevokeRoleCommand removes a role (by code) from a user.
// Revoking super_admin from its last holder is refused with ErrProtected.
type RevokeRoleCommand struct {
	UserID string
	Role   string // role code
}

// RevokeRole removes a role from a user. The last super_admin holder is
// protected, and the cache is invalidated after a successful revocation.
type RevokeRole struct {
	roles  domain.RoleRepository
	access domain.UserAccessRepository
	bumper CacheBumper
	audit  audit.Recorder
}

// NewRevokeRole builds the use case. bumper may be nil to skip cache
// invalidation and auditor may be nil to skip audit recording.
func NewRevokeRole(roles domain.RoleRepository, access domain.UserAccessRepository, bumper CacheBumper, auditor audit.Recorder) *RevokeRole {
	return &RevokeRole{roles: roles, access: access, bumper: bumper, audit: auditor}
}

// Execute runs the use case. It returns domain.ErrInvalid on blank user/role,
// domain.ErrNotFound when the role does not exist and domain.ErrProtected when
// the user is the last holder of super_admin. On success it bumps the cache
// (checked, one retry) and records an audit entry.
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
	if role.Code == authorization.RoleSuperAdmin {
		if err := uc.ensureSuperAdminRemains(ctx, userID, role); err != nil {
			return err
		}
	}
	if err := uc.access.RevokeRole(ctx, userID, role.ID); err != nil {
		return err
	}
	bumpChecked(ctx, uc.bumper)
	if uc.audit != nil {
		audit.RecordBestEffort(ctx, uc.audit, audit.Entry{
			SubjectType: "user_roles",
			SubjectID:   userID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"role_id": role.ID, "role": role.Code},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}

// ensureSuperAdminRemains rejects revoking super_admin from its last holder so
// an operator cannot lock everyone out of admin access.
func (uc *RevokeRole) ensureSuperAdminRemains(ctx context.Context, targetID string, role *domain.Role) error {
	roles, err := uc.access.Roles(ctx, targetID)
	if err != nil {
		return err
	}
	holds := false
	for _, r := range roles {
		if r == role.Code {
			holds = true
			break
		}
	}
	if !holds {
		return nil
	}
	n, err := uc.access.CountUsersWithRole(ctx, role.ID)
	if err != nil {
		return err
	}
	if n <= 1 {
		return domain.ErrProtected
	}
	return nil
}
