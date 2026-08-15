package command

import (
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
)

// isProtectedRole reports whether code refers to a built-in role that must not
// be renamed or deleted, otherwise the seeded guards and default assignment
// would break.
func isProtectedRole(code string) bool {
	switch code {
	case authorization.RoleSuperAdmin, authorization.RoleUser:
		return true
	}
	return false
}

// isProtectedPermission reports whether code refers to a built-in permission
// enforced by middleware constants, so renaming or deleting it would silently
// unlock protected endpoints.
func isProtectedPermission(code string) bool {
	return code == authorization.PermissionManageRBAC
}
