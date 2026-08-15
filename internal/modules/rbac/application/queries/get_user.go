// Package queries contains read-side use cases of the RBAC module.
package queries

import (
	"context"

	rbaccache "github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

// RolesPermissions is the effective set of a user's roles and permissions
// (direct grants plus everything inherited through roles).
type RolesPermissions struct {
	Roles       []string
	Permissions []string
}

// GetUser resolves a user's effective roles and permissions. Results are
// cached under the module's versioned permission cache when available.
type GetUser struct {
	access domain.UserAccessRepository
	cache  *rbaccache.PermissionCache
}

// NewGetUser builds the use case. cache may be nil to bypass caching.
func NewGetUser(access domain.UserAccessRepository, cache *rbaccache.PermissionCache) *GetUser {
	return &GetUser{access: access, cache: cache}
}

// Execute runs the use case.
func (q *GetUser) Execute(ctx context.Context, userID string) (*RolesPermissions, error) {
	if q.cache == nil {
		return q.load(ctx, userID)
	}
	ver, err := q.cache.CurrentVersion(ctx)
	if err != nil {
		return nil, err
	}
	roles, perms, ok, err := q.cache.GetUser(ctx, userID, ver)
	if err != nil {
		return nil, err
	}
	if ok {
		return &RolesPermissions{Roles: roles, Permissions: perms}, nil
	}
	res, err := q.load(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := q.cache.SetUser(ctx, userID, ver, res.Roles, res.Permissions); err != nil {
		return nil, err
	}
	return res, nil
}

func (q *GetUser) load(ctx context.Context, userID string) (*RolesPermissions, error) {
	roles, err := q.access.Roles(ctx, userID)
	if err != nil {
		return nil, err
	}
	direct, err := q.access.DirectPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	inherited, err := q.access.RolePermissionCodes(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &RolesPermissions{Roles: roles, Permissions: union(direct, inherited)}, nil
}

func union(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, s := range append(append([]string{}, a...), b...) {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
