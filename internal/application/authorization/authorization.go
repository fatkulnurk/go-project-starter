// Package authorization defines the cross-cutting authorization contract:
// "is the caller allowed to perform this action on this resource?".
package authorization

import (
	"context"
	"errors"
)

// ErrForbidden is returned by Can when the identity is not allowed.
var ErrForbidden = errors.New("forbidden")

// ErrAuthorizationDenied is an alias kept for readability at call sites.
var ErrAuthorizationDenied = ErrForbidden

// Authorizer answers "are you allowed to do this?".
type Authorizer interface {
	// Can checks whether identity may perform action on resource.
	Can(ctx context.Context, identity Identity, action string, resource any) error
	// HasRole checks whether identity holds the given role.
	HasRole(ctx context.Context, identity Identity, role string) error
}

// Identity is the minimal caller description needed for authorization.
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
	// PermissionManageMedia guards media management endpoints.
	PermissionManageMedia = "media.manage"
)

// AllowAll is an Authorizer that permits every request. It is the default for
// the starter until RBAC rules are introduced.
type AllowAll struct{}

// Can always returns nil.
func (AllowAll) Can(context.Context, Identity, string, any) error { return nil }

// HasRole always returns nil.
func (AllowAll) HasRole(context.Context, Identity, string) error { return nil }
