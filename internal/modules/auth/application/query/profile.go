// Package queries contains the read-side use cases of the auth module. The
// queries return read models built from the domain repositories.
package query

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/port"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// ProfileResult is the full profile of a user: their account plus the roles
// and permissions resolved from the RBAC module.
type ProfileResult struct {
	User        *domain.User
	Roles       []string
	Permissions []string
}

// Profile returns the profile of a user including their RBAC roles and
// permissions, resolved through the optional Roles port.
type Profile struct {
	users domain.UserRepository
	roles port.Roles
}

// NewProfile builds the profile use case. roles may be nil when RBAC is not
// wired; the returned profile then carries no roles or permissions.
func NewProfile(users domain.UserRepository, roles port.Roles) *Profile {
	return &Profile{users: users, roles: roles}
}

// Execute returns the user's profile together with RBAC data. It returns
// ErrNotFound when the user does not exist and passes through RBAC errors.
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
