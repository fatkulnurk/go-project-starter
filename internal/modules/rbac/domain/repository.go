package domain

import "context"

// RoleRepository persists roles and their permission links.
// Implementations are infrastructure-backed; this interface is what use cases
// depend on, keeping the domain free of any persistence library.
type RoleRepository interface {
	// Save inserts a new role. It returns an error when a role with the same
	// id already exists.
	Save(ctx context.Context, r *Role) error

	// FindByCode returns the role with code, or nil when it does not exist.
	// A nil, nil result means the code is not registered.
	FindByCode(ctx context.Context, code string) (*Role, error)

	// List returns all roles ordered by creation time, oldest first.
	// It is the backing read for role listing endpoints.
	List(ctx context.Context) ([]*Role, error)

	// Delete removes a role and its links.
	// Related user_roles/role_permissions rows are removed by ON DELETE CASCADE.
	Delete(ctx context.Context, id string) error

	// UpdateName renames the role's display label, keeping its users and
	// permission links intact. The code is immutable and never updated here.
	UpdateName(ctx context.Context, id, name string) error

	// SetPermissions replaces the role's permission set. The replacement must
	// be atomic (all-or-nothing).
	SetPermissions(ctx context.Context, roleID string, permissionIDs []string) error

	// PermissionsFor returns the permission codes granted to the role.
	// It is used by detail lookups; use PermissionsForAll for lists.
	PermissionsFor(ctx context.Context, roleID string) ([]string, error)

	// PermissionsForAll returns every role's permission codes in one query,
	// keyed by role id, so list endpoints avoid the N+1 pattern.
	PermissionsForAll(ctx context.Context) (map[string][]string, error)
}

// PermissionRepository persists permissions.
// Implementations are infrastructure-backed; use cases depend on this
// interface only, keeping the domain free of any persistence library.
type PermissionRepository interface {
	// Save inserts a new permission. It returns an error when a permission
	// with the same id already exists.
	Save(ctx context.Context, p *Permission) error

	// FindByCode returns the permission with code, or nil when it does not
	// exist.
	FindByCode(ctx context.Context, code string) (*Permission, error)

	// List returns all permissions ordered by group then code.
	// It is the backing read for permission listing endpoints.
	List(ctx context.Context) ([]*Permission, error)

	// Delete removes a permission and its links.
	// Related role_permissions/user_permissions rows are removed by ON DELETE
	// CASCADE.
	Delete(ctx context.Context, id string) error

	// Update renames the permission's display group and label, keeping its
	// code and links.
	Update(ctx context.Context, id, group, name string) error
}

// UserAccessRepository persists the user↔role and user↔permission links.
// Implementations guarantee idempotent grants so double-assignment is safe.
type UserAccessRepository interface {
	// AssignRole links the user to the role. The operation is idempotent:
	// granting an already-held role is a no-op.
	AssignRole(ctx context.Context, userID, roleID string) error

	// RevokeRole removes the user↔role link.
	// The operation is idempotent; revoking a role the user does not hold is
	// a no-op.
	RevokeRole(ctx context.Context, userID, roleID string) error

	// GrantPermission grants the permission directly to the user. The
	// operation is idempotent.
	GrantPermission(ctx context.Context, userID, permissionID string) error

	// RevokePermission removes the direct user↔permission link.
	// The operation is idempotent.
	RevokePermission(ctx context.Context, userID, permissionID string) error

	// Roles returns the role codes assigned to the user, ordered by code.
	// It is the role side of the user's effective permission resolution.
	Roles(ctx context.Context, userID string) ([]string, error)

	// CountUsersWithRole returns how many users currently hold the role. Used
	// to protect the last super_admin from being revoked.
	CountUsersWithRole(ctx context.Context, roleID string) (int, error)

	// DirectPermissions returns permission codes granted directly to the user
	// (not inherited through roles).
	DirectPermissions(ctx context.Context, userID string) ([]string, error)

	// RolePermissionCodes returns the permission codes inherited through the
	// user's roles.
	RolePermissionCodes(ctx context.Context, userID string) ([]string, error)
}
