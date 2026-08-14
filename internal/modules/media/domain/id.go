package domain

import (
	"crypto/rand"
	"encoding/hex"
	"path"
	"path/filepath"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/id"
)

// Storage key prefix for media objects.
const (
	keyPrefix     = "media"
	pathSeparator = "/"
	randTokenSize = 8
)

// newID returns a version-7 UUID string.
func newID() string { return id.New() }

// ObjectKey builds the storage key for a file:
// media/{modelType}/{modelID}/{collection}/{base-<rand>}{ext}. It rejects
// segments that could escape the media root (path traversal) or contain
// separators.
func ObjectKey(modelType, modelID, collection, name string) (string, error) {
	if err := validateSegment(modelType); err != nil {
		return "", err
	}
	if err := validateSegment(modelID); err != nil {
		return "", err
	}
	if collection == "" {
		collection = CollectionDefault
	}
	if err := validateSegment(collection); err != nil {
		return "", err
	}
	return path.Join(keyPrefix, modelType, modelID, collection, uniqueFileName(name)), nil
}

// validateSegment rejects empty, dot-relative, and separator-carrying values.
func validateSegment(s string) error {
	if s == "" {
		return ErrInvalid
	}
	if s == "." || s == ".." || strings.Contains(s, "/") || strings.Contains(s, "\\") || strings.Contains(s, "\x00") {
		return ErrInvalid
	}
	return nil
}

// uniqueFileName preserves the original extension but randomizes the stem so
// uploads never collide.
func uniqueFileName(name string) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	if ext == "" {
		ext = ".bin"
	}
	token := make([]byte, randTokenSize)
	if _, err := rand.Read(token); err != nil {
		panic("media: crypto/rand unavailable: " + err.Error())
	}
	safe := strings.ReplaceAll(strings.TrimSpace(stem), pathSeparator, "-")
	if safe == "" {
		safe = "file"
	}
	return safe + "-" + hex.EncodeToString(token) + ext
}
