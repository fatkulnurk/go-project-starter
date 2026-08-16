// Package cache defines the cross-cutting cache contract. Business modules
// depend on this interface only; the concrete driver (redis, memory, ...) is
// chosen in the composition root.
package cache

import (
	"context"
	"time"
)

// Cache is a simple key-value store with TTL.
// Values are opaque byte slices; entries expire after their TTL and are then
// reported as missing. Implementations (redis, memory, database) are chosen in
// the composition root.
type Cache interface {
	// Get returns the value for key; ErrNotFound when missing or expired.
	// A found value is always non-nil, so callers can distinguish a cache
	// miss from a stored empty payload.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores value under key for ttl (0 = no expiry). It replaces any
	// existing value and resets the TTL.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes key (no-op when missing). A subsequent Get reports the
	// key as missing until it is Set again.
	Delete(ctx context.Context, key string) error

	// GetDelete atomically returns the value for key and removes it in one
	// operation. It returns ErrNotFound when the key is missing or expired.
	// Used for single-use tokens that must not be redeemable twice.
	GetDelete(ctx context.Context, key string) ([]byte, error)

	// Increment increases the integer stored at key by delta, returns the new
	// value. Creates the key when missing, starting from delta.
	Increment(ctx context.Context, key string, delta int64) (int64, error)

	// Expire sets the TTL of an existing key. A non-positive ttl removes the
	// key entirely (same as Delete).
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// Ping reports whether the backing store is reachable. In-memory stores
	// always succeed.
	Ping(ctx context.Context) error

	// Close releases underlying resources. Database-backed caches that share
	// an application pool return nil without closing that pool.
	Close() error
}

// ErrNotFound is returned by Get when the key does not exist or has expired.
var ErrNotFound = &notFoundError{}

type notFoundError struct{}

// Error implements error for ErrNotFound, returning the stable message that
// callers can rely on when they need to render the text.
func (*notFoundError) Error() string { return "cache: key not found" }
