// Package storage provides storage driver implementations and a factory.
package storage

import (
	"fmt"

	"github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/fatkulnurk/go-project-starter/internal/platform/storage/local"
	"github.com/fatkulnurk/go-project-starter/internal/platform/storage/s3"
)

// New returns a storage.Storage for the configured driver.
func New(cfg config.StorageConfig) (storage.Storage, error) {
	switch cfg.Driver {
	case "s3":
		return s3.NewS3(cfg.S3)
	case "local", "":
		return local.NewLocal(cfg.Local), nil
	default:
		return nil, fmt.Errorf("unknown storage driver %q", cfg.Driver)
	}
}
