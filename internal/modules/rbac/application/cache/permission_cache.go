// Package cache implements versioned caching of a user's roles and
// permissions. A global version key is bumped on any RBAC change, invalidating
// stale user entries without needing to scan them.
package cache

import (
	"context"
	"encoding/json"
	"time"

	appcache "github.com/fatkulnurk/go-project-starter/internal/application/cache"
)

const (
	versionKey    = "rbac:ver"
	userKeyPrefix = "rbac:user:"
)

// PermissionCache stores and invalidates role/permission data in the shared
// cache (redis/memory).
type PermissionCache struct {
	c   appcache.Cache
	ttl time.Duration
}

// NewPermissionCache builds a permission cache.
func NewPermissionCache(c appcache.Cache, ttl time.Duration) *PermissionCache {
	return &PermissionCache{c: c, ttl: ttl}
}

// Bump increments the global version, invalidating all cached user entries.
func (pc *PermissionCache) Bump(ctx context.Context) error {
	_, err := pc.c.Increment(ctx, versionKey, 1)
	return err
}

// CurrentVersion returns the current version (0 when never bumped).
func (pc *PermissionCache) CurrentVersion(ctx context.Context) (int64, error) {
	b, err := pc.c.Get(ctx, versionKey)
	if err == appcache.ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var v int64
	if err := json.Unmarshal(b, &v); err != nil {
		return 0, err
	}
	return v, nil
}

// userEntry is what is stored per user.
type userEntry struct {
	Version     int64    `json:"version"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// GetUser returns the cached roles/permissions for userID at version ver.
// ok=false means the entry is missing or stale.
func (pc *PermissionCache) GetUser(ctx context.Context, userID string, ver int64) (roles, permissions []string, ok bool, err error) {
	b, err := pc.c.Get(ctx, userKeyPrefix+userID)
	if err == appcache.ErrNotFound {
		return nil, nil, false, nil
	}
	if err != nil {
		return nil, nil, false, err
	}
	var e userEntry
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, nil, false, err
	}
	if e.Version != ver {
		return nil, nil, false, nil
	}
	return e.Roles, e.Permissions, true, nil
}

// SetUser caches roles/permissions for userID at version ver.
func (pc *PermissionCache) SetUser(ctx context.Context, userID string, ver int64, roles, permissions []string) error {
	b, err := json.Marshal(userEntry{Version: ver, Roles: roles, Permissions: permissions})
	if err != nil {
		return err
	}
	return pc.c.Set(ctx, userKeyPrefix+userID, b, pc.ttl)
}
