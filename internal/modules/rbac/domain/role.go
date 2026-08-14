// Package domain holds the pure domain of the RBAC module: roles,
// permissions and their repository interfaces. No framework imports allowed.
package domain

import (
	"errors"

	"github.com/fatkulnurk/go-project-starter/internal/application/id"
)

// Errors returned by RBAC use cases.
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrInvalid  = errors.New("invalid")
)

// Role is a named set of permissions assigned to users.
type Role struct {
	ID        string
	Name      string
	CreatedAt int64
}

// NewRole builds a role from its name.
func NewRole(name string) (*Role, error) {
	if name == "" {
		return nil, ErrInvalid
	}
	return &Role{ID: newID(), Name: name}, nil
}

// Permission is a single capability that can be granted to users or roles.
type Permission struct {
	ID   string
	Name string
}

// NewPermission builds a permission from its name.
func NewPermission(name string) (*Permission, error) {
	if name == "" {
		return nil, ErrInvalid
	}
	return &Permission{ID: newID(), Name: name}, nil
}

func newID() string { return id.New() }
