package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// CreateRoleCommand creates a role from a stable code and a display name.
// The code is the immutable machine identifier checked by authorization.
type CreateRoleCommand struct {
	Code string
	Name string
}

// CreateRole persists a new role. It rejects blank codes/names and duplicate
// role codes, and records an audit entry when an auditor is configured.
type CreateRole struct {
	roles   domain.RoleRepository
	auditor audit.Recorder
}

// NewCreateRole builds the use case. auditor may be nil to skip audit
// recording.
func NewCreateRole(roles domain.RoleRepository, auditor audit.Recorder) *CreateRole {
	return &CreateRole{roles: roles, auditor: auditor}
}

// Execute runs the use case. cmd is trimmed before validation; it returns
// domain.ErrInvalid on blank code/name and domain.ErrConflict when the role
// code already exists. Repositories errors are propagated unchanged.
func (uc *CreateRole) Execute(ctx context.Context, cmd CreateRoleCommand) error {
	code := strings.TrimSpace(cmd.Code)
	name := strings.TrimSpace(cmd.Name)
	if code == "" || name == "" {
		return domain.ErrInvalid
	}
	existing, err := uc.roles.FindByCode(ctx, code)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.ErrConflict
	}
	role, err := domain.NewRole(code, name)
	if err != nil {
		return err
	}
	if err := uc.roles.Save(ctx, role); err != nil {
		return err
	}
	if uc.auditor != nil {
		audit.RecordBestEffort(ctx, uc.auditor, audit.Entry{
			SubjectType: "roles",
			SubjectID:   role.ID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"code": role.Code, "name": role.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
