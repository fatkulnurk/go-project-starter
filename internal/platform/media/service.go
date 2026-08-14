package media

import (
	"context"
	"errors"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	appmedia "github.com/fatkulnurk/go-project-starter/internal/application/media"
	"github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// Deps wires what the service needs. URLGenerator is optional: when nil (or
// the driver cannot expose URLs, e.g. local disk) URL returns ErrNoURL.
type Deps struct {
	Repo         mediaRepository
	Storage      storage.Storage
	URLGenerator storage.URLGenerator
	Disk         string
	Auditor      audit.Auditor
	Clock        clock.Clock
}

// Service implements appmedia.Library.
type Service struct {
	repo    mediaRepository
	store   storage.Storage
	urlGen  storage.URLGenerator
	disk    string
	auditor audit.Auditor
	clock   clock.Clock
}

// New builds the media service.
func New(deps Deps) *Service {
	return &Service{
		repo:    deps.Repo,
		store:   deps.Storage,
		urlGen:  deps.URLGenerator,
		disk:    deps.Disk,
		auditor: deps.Auditor,
		clock:   deps.Clock,
	}
}

// AddMedia implements appmedia.Library.
func (s *Service) AddMedia(ctx context.Context, in appmedia.AddMediaInput) (*appmedia.Media, error) {
	key, err := ObjectKey(in.ModelType, in.ModelID, in.Collection, in.Name)
	if err != nil {
		return nil, err
	}
	if err := s.store.Put(ctx, key, in.Reader); err != nil {
		return nil, err
	}

	m, err := newMedia(in, key, s.disk, s.clock.Now())
	if err != nil {
		_ = s.store.Delete(ctx, key)
		return nil, err
	}
	if err := s.repo.Save(ctx, m); err != nil {
		_ = s.store.Delete(ctx, key)
		return nil, err
	}
	if s.auditor != nil {
		_ = s.auditor.Record(ctx, audit.Entry{
			SubjectType: in.ModelType,
			SubjectID:   in.ModelID,
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

// GetMedia implements appmedia.Library.
func (s *Service) GetMedia(ctx context.Context, id string) (*appmedia.Media, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, appmedia.ErrNotFound
	}
	return m, nil
}

// ListByModel implements appmedia.Library.
func (s *Service) ListByModel(ctx context.Context, modelType, modelID, collection string) ([]*appmedia.Media, error) {
	return s.repo.ListByModel(ctx, modelType, modelID, collection)
}

// RemoveMedia implements appmedia.Library.
func (s *Service) RemoveMedia(ctx context.Context, id string) error {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return appmedia.ErrNotFound
	}
	if err := s.store.Delete(ctx, m.FileName); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, m.ID); err != nil {
		return err
	}
	if s.auditor != nil {
		_ = s.auditor.Record(ctx, audit.Entry{
			SubjectType: m.ModelType,
			SubjectID:   m.ModelID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"media_id": m.ID, "name": m.Name, "mime": m.MimeType, "size": m.Size, "disk": m.Disk},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}

// URL implements appmedia.Library.
func (s *Service) URL(ctx context.Context, id string) (string, error) {
	if s.urlGen == nil {
		return "", appmedia.ErrNoURL
	}
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return "", err
	}
	if m == nil {
		return "", appmedia.ErrNotFound
	}
	u, err := s.urlGen.URL(ctx, m.FileName)
	if err != nil {
		if errors.Is(err, storage.ErrNoURL) {
			return "", appmedia.ErrNoURL
		}
		return "", err
	}
	if u == "" {
		return "", appmedia.ErrNoURL
	}
	return u, nil
}

var _ appmedia.Library = (*Service)(nil)
