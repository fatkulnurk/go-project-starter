// Package queries contains read-side use cases of the auth module.
package queries

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/ports"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// ProfileResult is the full profile of a user: their account plus the roles
// and permissions resolved from the RBAC module.
type ProfileResult struct {
	User        *domain.User
	Roles       []string
	Permissions []string
}

// Profile returns the profile of a user including RBAC data.
type Profile struct {
	users domain.UserRepository
	roles ports.Roles
}

// NewProfile builds the use case. roles may be nil when RBAC is not wired.
func NewProfile(users domain.UserRepository, roles ports.Roles) *Profile {
	return &Profile{users: users, roles: roles}
}

// Execute runs the use case.
func (q *Profile) Execute(ctx context.Context, userID string) (*ProfileResult, error) {
	user, err := q.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrNotFound
	}
	res := &ProfileResult{User: user}
	if q.roles != nil {
		roles, permissions, err := q.roles.RolesAndPermissions(ctx, userID)
		if err != nil {
			return nil, err
		}
		res.Roles, res.Permissions = roles, permissions
	}
	return res, nil
}
