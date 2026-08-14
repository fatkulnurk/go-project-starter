package media

import (
	"time"

	appid "github.com/fatkulnurk/go-project-starter/internal/application/id"
	appmedia "github.com/fatkulnurk/go-project-starter/internal/application/media"
)

// newMedia builds a media record from the upload input, minting a time-ordered
// id and UTC timestamps.
func newMedia(in appmedia.AddMediaInput, fileName, disk string, now time.Time) (*appmedia.Media, error) {
	if in.ModelType == "" || in.ModelID == "" {
		return nil, appmedia.ErrInvalid
	}
	collection := in.Collection
	if collection == "" {
		collection = appmedia.CollectionDefault
	}
	name := in.Name
	if name == "" {
		name = fileName
	}
	mimeType := in.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	size := in.Size
	if size < 0 {
		size = 0
	}
	now = now.UTC()
	return &appmedia.Media{
		ID:             appid.New(),
		ModelType:      in.ModelType,
		ModelID:        in.ModelID,
		CollectionName: collection,
		Name:           name,
		FileName:       fileName,
		MimeType:       mimeType,
		Disk:           disk,
		Size:           size,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}
