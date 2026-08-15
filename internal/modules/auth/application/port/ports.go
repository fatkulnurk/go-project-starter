// Package ports defines the outbound dependencies of the auth module. Each
// port is a narrow interface implemented in the composition root, so the auth
// module never imports another module's internals.
package port

import "context"

// Roles is the RBAC port the auth module needs: read a user's roles and
// permissions, and assign the default role on registration.
type Roles interface {
	// RolesAndPermissions returns the effective role and permission names of
	// the user.
	RolesAndPermissions(ctx context.Context, userID string) (roles, permissions []string, err error)
	// AssignDefaultRole grants the default role (e.g. "user") to a fresh user.
	AssignDefaultRole(ctx context.Context, userID string) error
}
