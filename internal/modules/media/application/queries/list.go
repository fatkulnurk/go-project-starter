package queries

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/modules/media/domain"
)

// ListByModelQuery filters media by model and collection.
type ListByModelQuery struct {
	ModelType  string
	ModelID    string
	Collection string
}

// ListByModel returns the media attached to a model.
type ListByModel struct {
	media domain.MediaRepository
}

// NewListByModel builds the use case.
func NewListByModel(media domain.MediaRepository) *ListByModel {
	return &ListByModel{media: media}
}

// Execute runs the use case.
func (q *ListByModel) Execute(ctx context.Context, query ListByModelQuery) ([]*domain.Media, error) {
	return q.media.ListByModel(ctx, query.ModelType, query.ModelID, query.Collection)
}
