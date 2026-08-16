// Package authorization defines the cross-cutting authorization contract:
// "is the caller allowed to perform this action on this resource?".
package authorization

import (
	"context"
	"errors"
)

// ErrForbidden is returned by HasPermission when the identity is not allowed.
var ErrForbidden = errors.New("forbidden")

// Authorizer answers "are you allowed to do this?". The RBAC module implements
// it; protected routes receive it via middleware.
type Authorizer interface {
	// HasPermission checks whether identity holds the given permission. It
	// returns nil when allowed and ErrForbidden when not.
	HasPermission(ctx context.Context, identity Identity, permission string) error

	// HasRole checks whether identity holds the given role. It returns nil
	// when allowed and ErrForbidden when not.
	HasRole(ctx context.Context, identity Identity, role string) error
}

// Identity is the minimal caller description needed for authorization: their
// user id and the roles asserted at authentication time.
type Identity struct {
	UserID string
	Roles  []string
}

// Well-known role names. RBAC seeds these and the auth module assigns the
// default role, so the constants never drift from what is stored.
const (
	RoleSuperAdmin = "super_admin"
	RoleUser       = "user"
)

// Well-known permission names. RBAC seeds these and protected modules check
// the same constants, so the names never drift from what is stored.
const (
	// PermissionManageRBAC guards the RBAC admin API (roles, permissions,
	// assignments).
	PermissionManageRBAC = "rbac.manage"
)
