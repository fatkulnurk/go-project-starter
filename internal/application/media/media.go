// Package media defines the cross-cutting media contract: files attached to
// any model (Laravel media-library style). Business modules depend on this
// interface only; the concrete implementation (database + storage driver)
// lives in internal/platform/media.
package media

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/apierr"
)

// Errors returned by the media contract.
var (
	ErrNotFound = apierr.ErrNotFound
	ErrInvalid  = apierr.ErrInvalid
	// ErrNoURL is returned by URL when the backing storage cannot expose a
	// public URL (e.g. the local filesystem has no HTTP server).
	ErrNoURL = errors.New("media: no public URL for object")
)

// Well-known collection names.
const (
	CollectionDefault = "default"
	CollectionAvatar  = "avatar"
)

// Media is a file attached to a model: the metadata row plus the storage key
// of the actual object. It is the value returned by Library operations.
type Media struct {
	ID             string
	ModelType      string
	ModelID        string
	CollectionName string
	Name           string // original user-facing name, e.g. "report.pdf"
	FileName       string // storage key relative to the disk root
	MimeType       string
	Disk           string
	Size           int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AddMediaInput carries what is needed to attach a new file: which model and
// collection it belongs to, its display name and MIME type, and the content.
type AddMediaInput struct {
	ModelType  string
	ModelID    string
	Collection string
	Name       string
	MimeType   string
	Size       int64
	Reader     io.Reader
}

// Library is the programmatic surface for attaching, reading and removing
// media. It is implemented by internal/platform/media and injectable into any
// module or adapter.
type Library interface {
	// AddMedia stores the file under a generated key and registers the media
	// row for the given model/collection. It returns the created record.
	AddMedia(ctx context.Context, in AddMediaInput) (*Media, error)

	// GetMedia returns the metadata of one record; ErrNotFound when missing.
	// It does not read the stored object; use URL to build a fetchable link.
	GetMedia(ctx context.Context, id string) (*Media, error)

	// ListByModel returns the media attached to a model, optionally filtered
	// by collection (empty = every collection).
	ListByModel(ctx context.Context, modelType, modelID, collection string) ([]*Media, error)

	// RemoveMedia deletes the object from storage and the database row. It is
	// a no-op when the record does not exist.
	RemoveMedia(ctx context.Context, id string) error

	// URL returns a public URL for the media's stored object; ErrNoURL when
	// the storage driver cannot expose one.
	URL(ctx context.Context, id string) (string, error)
}
