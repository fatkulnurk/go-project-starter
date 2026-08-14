package httpapi

import (
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/modules/media/domain"
)

// Paths, form fields, query params and headers, as constants so no magic
// strings appear in handlers.
const (
	mediaBasePath            = "/api/v1/media"
	pathDownload             = "/download"
	formFieldFile            = "file"
	formFieldModelType       = "model_type"
	formFieldModelID         = "model_id"
	formFieldCollection      = "collection"
	queryModelType           = "model_type"
	queryModelID             = "model_id"
	queryCollection          = "collection"
	headerContentType        = "Content-Type"
	headerContentLength      = "Content-Length"
	headerContentDisposition = "Content-Disposition"
	maxUploadMemory          = 32 << 20
)

type mediaResponse struct {
	ID             string    `json:"id"`
	ModelType      string    `json:"model_type"`
	ModelID        string    `json:"model_id"`
	CollectionName string    `json:"collection_name"`
	Name           string    `json:"name"`
	FileName       string    `json:"file_name"`
	MimeType       string    `json:"mime_type"`
	Disk           string    `json:"disk"`
	Size           int64     `json:"size"`
	URL            string    `json:"url"`
	CreatedAt      time.Time `json:"created_at"`
}

func toMediaResponse(m *domain.Media, url string) mediaResponse {
	return mediaResponse{
		ID:             m.ID,
		ModelType:      m.ModelType,
		ModelID:        m.ModelID,
		CollectionName: m.CollectionName,
		Name:           m.Name,
		FileName:       m.FileName,
		MimeType:       m.MimeType,
		Disk:           m.Disk,
		Size:           m.Size,
		URL:            url,
		CreatedAt:      m.CreatedAt,
	}
}

func toMediaResponses(items []*domain.Media, url func(*domain.Media) string) []mediaResponse {
	out := make([]mediaResponse, 0, len(items))
	for _, m := range items {
		out = append(out, toMediaResponse(m, url(m)))
	}
	return out
}
