package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// CreateRoleCommand creates a named role.
type CreateRoleCommand struct {
	Name string
}

// CreateRole persists a new role.
type CreateRole struct {
	roles   domain.RoleRepository
	auditor audit.Auditor
}

// NewCreateRole builds the use case.
func NewCreateRole(roles domain.RoleRepository, auditor audit.Auditor) *CreateRole {
	return &CreateRole{roles: roles, auditor: auditor}
}

// Execute runs the use case.
func (uc *CreateRole) Execute(ctx context.Context, cmd CreateRoleCommand) error {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return domain.ErrInvalid
	}
	existing, err := uc.roles.FindByName(ctx, name)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.ErrConflict
	}
	role, err := domain.NewRole(name)
	if err != nil {
		return err
	}
	if err := uc.roles.Save(ctx, role); err != nil {
		return err
	}
	if uc.auditor != nil {
		_ = uc.auditor.Record(ctx, audit.Entry{
			SubjectType: "roles",
			SubjectID:   role.ID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"name": role.Name},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
