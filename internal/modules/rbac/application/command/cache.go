package command

import "context"

// CacheBumper invalidates cached RBAC data after a mutation. It is implemented
// by the module's permission cache; nil disables caching (no-op).
type CacheBumper interface {
	Bump(ctx context.Context) error
}

// bumpBestEffort invalidates the RBAC cache after a committed mutation. A bump
// failure must never fail a request whose database change is already durable,
// otherwise clients would retry an already-applied mutation and hit a
// confusing conflict. Stale entries self-heal via their TTL, so the cost of a
// failed bump is bounded staleness, not corruption.
func bumpBestEffort(ctx context.Context, b CacheBumper) {
	if b == nil {
		return
	}
	_ = b.Bump(ctx)
}
