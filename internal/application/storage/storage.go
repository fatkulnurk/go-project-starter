// Package storage defines the cross-cutting storage contract. Business
// modules only know Storage/Presigner; the concrete driver (local, s3, ...)
// is chosen in the composition root.
package storage

import (
	"context"
	"errors"
	"io"
)

// ErrNotFound is returned by Get/Attributes when the object does not exist.
var ErrNotFound = errors.New("storage: object not found")

// ErrInvalidKey is returned when a key is rejected (empty or path traversal).
var ErrInvalidKey = errors.New("storage: invalid object key")

// ErrNoURL is returned by URLGenerator when the driver cannot produce a
// public URL for an object (e.g. the local filesystem has no HTTP server).
var ErrNoURL = errors.New("storage: no public URL for object")

// ObjectAttrs carries metadata about a stored object.
type ObjectAttrs struct {
	Size int64
}

// Storage is a key-based object store (local filesystem, S3, S3-compatible).
type Storage interface {
	// Put stores the object under key.
	Put(ctx context.Context, key string, r io.Reader) error
	// Get returns the object body for key; ErrNotFound when missing.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the object under key (no-op when missing).
	Delete(ctx context.Context, key string) error
	// Attributes returns metadata for key; ErrNotFound when missing.
	Attributes(ctx context.Context, key string) (ObjectAttrs, error)
}

// Presigner is optional; implemented by drivers that can produce signed URLs
// (e.g. private objects over HTTP).
type Presigner interface {
	// Presign returns a time-limited URL to GET key.
	Presign(ctx context.Context, key string) (string, error)
}

// URLGenerator is optional; implemented by drivers that can produce a public
// URL for a stored object. Drivers without public serving (e.g. local
// filesystem) return ErrNoURL.
type URLGenerator interface {
	// URL returns a public URL to GET key. ErrNoURL when the driver cannot
	// produce one.
	URL(ctx context.Context, key string) (string, error)
}
