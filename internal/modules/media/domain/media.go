// Package domain holds the pure domain of the media module: the Media entity
// and its repository interface. No framework imports allowed.
package domain

import (
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/apierr"
)

// Errors returned by media use cases.
var (
	ErrNotFound = apierr.ErrNotFound
	ErrInvalid  = apierr.ErrInvalid
)

// Well-known collection names.
const (
	CollectionDefault = "default"
	CollectionAvatar  = "avatar"
)

// Media is a file attached to a model, Laravel media-library style.
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

// NewMedia builds a media record.
func NewMedia(modelType, modelID, collection, name, fileName, mimeType, disk string, size int64, now time.Time) (*Media, error) {
	if modelType == "" || modelID == "" {
		return nil, ErrInvalid
	}
	if collection == "" {
		collection = CollectionDefault
	}
	if name == "" {
		name = fileName
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if size < 0 {
		size = 0
	}
	now = now.UTC()
	return &Media{
		ID:             newID(),
		ModelType:      modelType,
		ModelID:        modelID,
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
