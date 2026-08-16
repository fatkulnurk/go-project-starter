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

// ObjectAttrs carries metadata about a stored object, currently its size in
// bytes. New fields are additive so callers stay source-compatible.
type ObjectAttrs struct {
	Size int64
}

// Storage is a key-based object store (local filesystem, S3, S3-compatible).
// Keys are opaque to callers but must be driver-safe; invalid keys are
// rejected with ErrInvalidKey.
type Storage interface {
	// Put stores the object under key, replacing any existing object. The
	// reader is consumed; the object is fully written before returning.
	Put(ctx context.Context, key string, r io.Reader) error

	// Get returns the object body for key; ErrNotFound when missing. Callers
	// must Close the returned reader to release resources.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes the object under key (no-op when missing). It returns an
	// error only when the backend cannot perform the removal.
	Delete(ctx context.Context, key string) error

	// Attributes returns metadata for key; ErrNotFound when missing. It does
	// not stream the object body.
	Attributes(ctx context.Context, key string) (ObjectAttrs, error)
}

// Presigner is optional; implemented by drivers that can produce signed URLs
// (e.g. private objects over HTTP). Type-assert Storage to Presigner to use it.
type Presigner interface {
	// Presign returns a time-limited URL to GET key. The URL expires and must
	// be fetched before the driver's configured validity window.
	Presign(ctx context.Context, key string) (string, error)
}

// URLGenerator is optional; implemented by drivers that can produce a public
// URL for a stored object. Drivers without public serving (e.g. local
// filesystem) return ErrNoURL.
type URLGenerator interface {
	// URL returns a public URL to GET key. ErrNoURL when the driver cannot
	// produce one (e.g. local storage without an HTTP endpoint).
	URL(ctx context.Context, key string) (string, error)
}
