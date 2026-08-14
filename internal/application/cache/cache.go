// Package cache defines the cross-cutting cache contract. Business modules
// depend on this interface only; the concrete driver (redis, memory, ...) is
// chosen in the composition root.
package cache

import (
	"context"
	"time"
)

// Cache is a simple key-value store with TTL.
type Cache interface {
	// Get returns the value for key; ErrNotFound when missing or expired.
	Get(ctx context.Context, key string) ([]byte, error)
	// Set stores value under key for ttl (0 = no expiry).
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Delete removes key (no-op when missing).
	Delete(ctx context.Context, key string) error
	// Increment increases the integer stored at key by delta, returns the new
	// value. Creates the key when missing.
	Increment(ctx context.Context, key string, delta int64) (int64, error)
	// Expire sets the TTL of an existing key.
	Expire(ctx context.Context, key string, ttl time.Duration) error
	// Ping reports whether the backing store is reachable. In-memory stores
	// always succeed.
	Ping(ctx context.Context) error
	// Close releases underlying resources.
	Close() error
}

// ErrNotFound is returned by Get when the key does not exist or has expired.
var ErrNotFound = &notFoundError{}

type notFoundError struct{}

func (*notFoundError) Error() string { return "cache: key not found" }
