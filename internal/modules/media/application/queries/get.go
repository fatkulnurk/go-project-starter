// Package queries contains the read-side use cases of the media module.
package queries

import (
	"context"
	"errors"
	"io"

	"github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/domain"
)

// GetMediaResult carries the media metadata and an open reader for its bytes.
type GetMediaResult struct {
	Media  *domain.Media
	Reader io.ReadCloser
}

// GetMedia returns a media record and its file stream.
type GetMedia struct {
	media   domain.MediaRepository
	storage storage.Storage
}

// NewGetMedia builds the use case.
func NewGetMedia(media domain.MediaRepository, storage storage.Storage) *GetMedia {
	return &GetMedia{media: media, storage: storage}
}

// Execute runs the use case.
func (q *GetMedia) Execute(ctx context.Context, id string) (*GetMediaResult, error) {
	m, err := q.media.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, domain.ErrNotFound
	}
	rc, err := q.storage.Get(ctx, m.FileName)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &GetMediaResult{Media: m, Reader: rc}, nil
}
