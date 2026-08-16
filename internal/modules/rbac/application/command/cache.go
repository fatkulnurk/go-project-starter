package command

import (
	"context"
	"log/slog"
)

// CacheBumper invalidates cached RBAC data after a mutation. It is implemented
// by the module's permission cache; nil disables caching (no-op).
type CacheBumper interface {
	// Bump invalidates all cached RBAC entries by advancing the global
	// version. It returns an error when the cache increment fails.
	Bump(ctx context.Context) error
}

// bumpBestEffort invalidates the RBAC cache after a committed mutation. A bump
// failure must never fail a request whose database change is already durable,
// otherwise clients would retry an already-applied mutation and hit a
// confusing conflict. Stale entries self-heal via their TTL, so the cost of a
// failed bump is bounded staleness, not corruption. Failures are logged so
// they are not silent.
func bumpBestEffort(ctx context.Context, b CacheBumper) {
	if b == nil {
		return
	}
	if err := b.Bump(ctx); err != nil {
		slog.Warn("rbac cache bump failed; stale permissions until TTL", "err", err)
	}
}

// bumpChecked invalidates the RBAC cache after a mutation that must propagate
// quickly (role/permission assignment or revocation). It retries once before
// giving up, because a stale cache after a revoke can keep granting access
// that was just removed. The request still succeeds when both attempts fail;
// the staleness window is bounded by the cache TTL.
func bumpChecked(ctx context.Context, b CacheBumper) {
	if b == nil {
		return
	}
	if err := b.Bump(ctx); err != nil {
		if err2 := b.Bump(ctx); err2 != nil {
			slog.Error("rbac cache bump failed after retry; access change may persist until TTL", "err", err2)
		}
	}
}
