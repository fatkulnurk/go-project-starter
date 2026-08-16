package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// AssignRoleCommand assigns a role (by code) to a user.
// The role code is resolved to its id; unknown roles fail with ErrNotFound.
type AssignRoleCommand struct {
	UserID string
	Role   string // role code
}

// AssignRole grants a role to a user. Super_admin grants are guarded against
// privilege escalation, and the cache is invalidated after a successful grant.
type AssignRole struct {
	roles  domain.RoleRepository
	access domain.UserAccessRepository
	bumper CacheBumper
	audit  audit.Recorder
}

// NewAssignRole builds the use case. bumper may be nil to skip cache
// invalidation and auditor may be nil to skip audit recording.
func NewAssignRole(roles domain.RoleRepository, access domain.UserAccessRepository, bumper CacheBumper, auditor audit.Recorder) *AssignRole {
	return &AssignRole{roles: roles, access: access, bumper: bumper, audit: auditor}
}

// Execute runs the use case. It returns domain.ErrInvalid on blank user/role,
// domain.ErrNotFound when the role does not exist and domain.ErrProtected when
// the grant is a privilege-escalating super_admin assignment. On success it
// bumps the cache (checked, one retry) and records an audit entry.
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
	if role.Code == authorization.RoleSuperAdmin {
		if err := uc.authorizeSuperAdminGrant(ctx, userID); err != nil {
			return err
		}
	}
	if err := uc.access.AssignRole(ctx, userID, role.ID); err != nil {
		return err
	}
	bumpChecked(ctx, uc.bumper)
	if uc.audit != nil {
		audit.RecordBestEffort(ctx, uc.audit, audit.Entry{
			SubjectType: "user_roles",
			SubjectID:   userID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"role_id": role.ID, "role": role.Code},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}

// authorizeSuperAdminGrant prevents privilege escalation: a caller cannot
// grant super_admin to themselves, and granting it to someone else requires
// the caller to already hold super_admin. Trusted internal calls (the seeder
// or bootstrap, which carry no identity) are allowed.
func (uc *AssignRole) authorizeSuperAdminGrant(ctx context.Context, targetID string) error {
	id := appauth.IdentityFrom(ctx)
	if id == nil {
		return nil
	}
	if id.UserID == targetID {
		return domain.ErrProtected
	}
	roles, err := uc.access.Roles(ctx, id.UserID)
	if err != nil {
		return err
	}
	for _, r := range roles {
		if r == authorization.RoleSuperAdmin {
			return nil
		}
	}
	return domain.ErrProtected
}
