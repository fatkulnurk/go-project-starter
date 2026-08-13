package domain

import "context"

// MediaRepository persists media records.
type MediaRepository interface {
	Save(ctx context.Context, m *Media) error
	FindByID(ctx context.Context, id string) (*Media, error)
	ListByModel(ctx context.Context, modelType, modelID, collection string) ([]*Media, error)
	Delete(ctx context.Context, id string) error
}
