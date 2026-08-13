package domain

import "context"

// RoleRepository persists roles and their permission links.
type RoleRepository interface {
	Save(ctx context.Context, r *Role) error
	FindByName(ctx context.Context, name string) (*Role, error)
	List(ctx context.Context) ([]*Role, error)
	// Delete removes a role and its links.
	Delete(ctx context.Context, id string) error
	// SetPermissions replaces the role's permission set.
	SetPermissions(ctx context.Context, roleID string, permissionIDs []string) error
	// PermissionsFor returns the permission names granted to the role.
	PermissionsFor(ctx context.Context, roleID string) ([]string, error)
}

// PermissionRepository persists permissions.
type PermissionRepository interface {
	Save(ctx context.Context, p *Permission) error
	FindByName(ctx context.Context, name string) (*Permission, error)
	List(ctx context.Context) ([]*Permission, error)
}

// UserAccessRepository persists the user↔role and user↔permission links.
type UserAccessRepository interface {
	AssignRole(ctx context.Context, userID, roleID string) error
	RevokeRole(ctx context.Context, userID, roleID string) error
	GrantPermission(ctx context.Context, userID, permissionID string) error
	RevokePermission(ctx context.Context, userID, permissionID string) error
	// Roles returns the role names assigned to the user.
	Roles(ctx context.Context, userID string) ([]string, error)
	// DirectPermissions returns permission names granted directly to the user
	// (not inherited through roles).
	DirectPermissions(ctx context.Context, userID string) ([]string, error)
	// RolePermissionNames returns the permission names inherited through the
	// user's roles.
	RolePermissionNames(ctx context.Context, userID string) ([]string, error)
}
