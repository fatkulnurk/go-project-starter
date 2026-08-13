package commands

import "context"

// CacheBumper invalidates cached RBAC data after a mutation. It is implemented
// by the module's permission cache; nil disables caching (no-op).
type CacheBumper interface {
	Bump(ctx context.Context) error
}

func bump(ctx context.Context, b CacheBumper) error {
	if b == nil {
		return nil
	}
	return b.Bump(ctx)
}
