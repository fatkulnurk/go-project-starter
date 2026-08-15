// Package domain holds the pure domain of the RBAC module: roles,
// permissions and their repository interfaces. No framework imports allowed.
package domain

import (
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/apierr"
	"github.com/fatkulnurk/go-project-starter/internal/application/id"
)

// Errors returned by RBAC use cases. They carry their HTTP kind so the API
// layer renders the correct status code.
var (
	ErrNotFound  = apierr.New(apierr.KindNotFound, "not found")
	ErrConflict  = apierr.New(apierr.KindConflict, "conflict")
	ErrInvalid   = apierr.New(apierr.KindInvalid, "invalid")
	ErrProtected = apierr.New(apierr.KindForbidden, "protected role or permission")
)

// Role is a named set of permissions assigned to users. Code is the stable
// machine identifier checked by authorization; Name is the display label.
type Role struct {
	ID        string
	Code      string
	Name      string
	CreatedAt int64
}

// NewRole builds a role from its stable code and display name.
func NewRole(code, name string) (*Role, error) {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" {
		return nil, ErrInvalid
	}
	return &Role{ID: newID(), Code: code, Name: name}, nil
}

// Permission is a single capability that can be granted to users or roles.
// Code is the stable identifier checked by authorization; Group and Name are
// display metadata.
type Permission struct {
	ID    string
	Code  string
	Group string
	Name  string
}

// NewPermission builds a permission from its stable code, display group and
// display name.
func NewPermission(code, group, name string) (*Permission, error) {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" {
		return nil, ErrInvalid
	}
	if strings.TrimSpace(group) == "" {
		group = "General"
	}
	return &Permission{ID: newID(), Code: code, Group: group, Name: name}, nil
}

func newID() string { return id.New() }
