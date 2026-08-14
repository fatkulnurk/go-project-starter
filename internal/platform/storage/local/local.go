// Package local implements the application/storage contract on the local
// filesystem. Files are stored under a configurable root directory.
package local

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// Local is a filesystem-backed storage.
type Local struct {
	baseDir string
}

// NewLocal builds a local storage rooted at cfg.Dir.
func NewLocal(cfg config.LocalStorageConfig) *Local {
	return &Local{baseDir: filepath.Clean(cfg.Dir)}
}

// resolve joins key under baseDir and verifies the result stays inside baseDir,
// rejecting path traversal such as "../" or absolute keys.
func (s *Local) resolve(key string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(key))
	base := filepath.Clean(s.baseDir)
	full := filepath.Join(base, cleaned)
	rel, err := filepath.Rel(base, full)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", storage.ErrInvalidKey
	}
	return full, nil
}

// Put implements storage.Storage.
func (s *Local) Put(_ context.Context, key string, r io.Reader) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
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
	path, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, storage.ErrNotFound
	}
	return f, err
}

// Delete implements storage.Storage.
func (s *Local) Delete(_ context.Context, key string) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Attributes implements storage.Storage.
func (s *Local) Attributes(_ context.Context, key string) (storage.ObjectAttrs, error) {
	path, err := s.resolve(key)
	if err != nil {
		return storage.ObjectAttrs{}, err
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return storage.ObjectAttrs{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.ObjectAttrs{}, err
	}
	return storage.ObjectAttrs{Size: info.Size()}, nil
}

// URL implements storage.URLGenerator. The local driver has no HTTP server of
// its own, so it cannot expose a direct public URL; callers must fall back to
// a proxying endpoint.
func (s *Local) URL(context.Context, string) (string, error) {
	return "", storage.ErrNoURL
}
