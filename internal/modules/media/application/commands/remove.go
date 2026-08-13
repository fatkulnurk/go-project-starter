package commands

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/domain"
)

// RemoveMediaCommand deletes a media record and its underlying object.
type RemoveMediaCommand struct {
	ID string
}

// RemoveMedia deletes the object from storage and the row from the database.
type RemoveMedia struct {
	media   domain.MediaRepository
	storage storage.Storage
}

// NewRemoveMedia builds the use case.
func NewRemoveMedia(media domain.MediaRepository, storage storage.Storage) *RemoveMedia {
	return &RemoveMedia{media: media, storage: storage}
}

// Execute runs the use case.
func (uc *RemoveMedia) Execute(ctx context.Context, cmd RemoveMediaCommand) error {
	m, err := uc.media.FindByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if m == nil {
		return domain.ErrNotFound
	}
	if err := uc.storage.Delete(ctx, m.FileName); err != nil {
		return err
	}
	return uc.media.Delete(ctx, m.ID)
}
