package query

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// FindUserByEmailResult is the minimal user data needed by other modules
// (e.g. RBAC bootstrap resolves a user by email to grant the super admin).
type FindUserByEmailResult struct {
	ID    string
	Email string
	Name  string
}

// FindUserByEmail looks up a user by email.
type FindUserByEmail struct {
	users domain.UserRepository
}

// NewFindUserByEmail builds the use case.
func NewFindUserByEmail(users domain.UserRepository) *FindUserByEmail {
	return &FindUserByEmail{users: users}
}

// Execute runs the use case.
func (q *FindUserByEmail) Execute(ctx context.Context, email string) (*FindUserByEmailResult, error) {
	user, err := q.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrNotFound
	}
	emailAddr := ""
	if user.Email != nil {
		emailAddr = *user.Email
	}
	return &FindUserByEmailResult{ID: user.ID, Email: emailAddr, Name: user.Name}, nil
}
