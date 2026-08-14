// Package commands contains the write-side use cases of the media module.
package commands

import (
	"context"
	"io"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// AddMediaCommand uploads a file and registers it for a model.
type AddMediaCommand struct {
	ModelType  string
	ModelID    string
	Collection string
	Name       string
	MimeType   string
	Size       int64
	Reader     io.Reader
}

// AddMedia stores the file and its metadata. The object key is derived from
// the model; the storage key is kept in FileName.
type AddMedia struct {
	media   domain.MediaRepository
	storage storage.Storage
	disk    string
	auditor audit.Auditor
	clock   clock.Clock
}

// NewAddMedia builds the use case.
func NewAddMedia(media domain.MediaRepository, storage storage.Storage, disk string, auditor audit.Auditor, clk clock.Clock) *AddMedia {
	return &AddMedia{media: media, storage: storage, disk: disk, auditor: auditor, clock: clk}
}

// Execute runs the use case.
func (uc *AddMedia) Execute(ctx context.Context, cmd AddMediaCommand) (*domain.Media, error) {
	key, err := domain.ObjectKey(cmd.ModelType, cmd.ModelID, cmd.Collection, cmd.Name)
	if err != nil {
		return nil, err
	}
	if err := uc.storage.Put(ctx, key, cmd.Reader); err != nil {
		return nil, err
	}

	m, err := domain.NewMedia(cmd.ModelType, cmd.ModelID, cmd.Collection, cmd.Name, key, cmd.MimeType, uc.disk, cmd.Size, uc.clock.Now())
	if err != nil {
		_ = uc.storage.Delete(ctx, key)
		return nil, err
	}
	if err := uc.media.Save(ctx, m); err != nil {
		_ = uc.storage.Delete(ctx, key)
		return nil, err
	}
	if uc.auditor != nil {
		_ = uc.auditor.Record(ctx, audit.Entry{
			SubjectType: cmd.ModelType,
			SubjectID:   cmd.ModelID,
			Action:      audit.ActionCreated,
			NewValues: map[string]any{
				"media_id": m.ID,
				"name":     m.Name,
				"mime":     m.MimeType,
				"size":     m.Size,
				"disk":     m.Disk,
			},
			Actor: audit.ActorFrom(ctx),
		})
	}
	return m, nil
}
