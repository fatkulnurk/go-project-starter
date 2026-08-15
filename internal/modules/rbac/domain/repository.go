package domain

import "context"

// RoleRepository persists roles and their permission links.
type RoleRepository interface {
	Save(ctx context.Context, r *Role) error
	FindByCode(ctx context.Context, code string) (*Role, error)
	List(ctx context.Context) ([]*Role, error)
	// Delete removes a role and its links.
	Delete(ctx context.Context, id string) error
	// UpdateName renames the role's display label, keeping its links.
	UpdateName(ctx context.Context, id, name string) error
	// SetPermissions replaces the role's permission set.
	SetPermissions(ctx context.Context, roleID string, permissionIDs []string) error
	// PermissionsFor returns the permission codes granted to the role.
	PermissionsFor(ctx context.Context, roleID string) ([]string, error)
}

// PermissionRepository persists permissions.
type PermissionRepository interface {
	Save(ctx context.Context, p *Permission) error
	FindByCode(ctx context.Context, code string) (*Permission, error)
	List(ctx context.Context) ([]*Permission, error)
	// Delete removes a permission and its links.
	Delete(ctx context.Context, id string) error
	// Update renames the permission's display group and label, keeping its
	// code and links.
	Update(ctx context.Context, id, group, name string) error
}

// UserAccessRepository persists the user↔role and user↔permission links.
type UserAccessRepository interface {
	AssignRole(ctx context.Context, userID, roleID string) error
	RevokeRole(ctx context.Context, userID, roleID string) error
	GrantPermission(ctx context.Context, userID, permissionID string) error
	RevokePermission(ctx context.Context, userID, permissionID string) error
	// Roles returns the role codes assigned to the user.
	Roles(ctx context.Context, userID string) ([]string, error)
	// DirectPermissions returns permission codes granted directly to the user
	// (not inherited through roles).
	DirectPermissions(ctx context.Context, userID string) ([]string, error)
	// RolePermissionCodes returns the permission codes inherited through the
	// user's roles.
	RolePermissionCodes(ctx context.Context, userID string) ([]string, error)
}
