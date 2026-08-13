// Package local implements the application/storage contract on the local
// filesystem. Files are stored under a configurable root directory.
package local

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// Local is a filesystem-backed storage.
type Local struct {
	baseDir string
}

// NewLocal builds a local storage rooted at cfg.Dir.
func NewLocal(cfg config.LocalStorageConfig) *Local {
	return &Local{baseDir: cfg.Dir}
}

// Put implements storage.Storage.
func (s *Local) Put(_ context.Context, key string, r io.Reader) error {
	path := filepath.Join(s.baseDir, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

// Get implements storage.Storage.
func (s *Local) Get(_ context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(filepath.Join(s.baseDir, key))
	if os.IsNotExist(err) {
		return nil, storage.ErrNotFound
	}
	return f, err
}

// Delete implements storage.Storage.
func (s *Local) Delete(_ context.Context, key string) error {
	err := os.Remove(filepath.Join(s.baseDir, key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Attributes implements storage.Storage.
func (s *Local) Attributes(_ context.Context, key string) (storage.ObjectAttrs, error) {
	info, err := os.Stat(filepath.Join(s.baseDir, key))
	if os.IsNotExist(err) {
		return storage.ObjectAttrs{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.ObjectAttrs{}, err
	}
	return storage.ObjectAttrs{Size: info.Size()}, nil
}
