// Package media implements the application/media contract: metadata is
// persisted in the media table and file bytes live behind the application
// storage driver (local, s3).
package media

import (
	"crypto/rand"
	"encoding/hex"
	"path"
	"path/filepath"
	"strings"

	appmedia "github.com/fatkulnurk/go-project-starter/internal/application/media"
)

// Storage key prefix for media objects.
const (
	keyPrefix     = "media"
	randTokenSize = 8
)

// ObjectKey builds the storage key for a file:
// media/{modelType}/{modelID}/{collection}/{base-<rand>}{ext}. It rejects
// segments that could escape the media root (path traversal) or contain
// separators.
func ObjectKey(modelType, modelID, collection, name string) (string, error) {
	if err := validateSegment(modelType); err != nil {
		return "", appmedia.ErrInvalid
	}
	if err := validateSegment(modelID); err != nil {
		return "", appmedia.ErrInvalid
	}
	if collection == "" {
		collection = appmedia.CollectionDefault
	}
	if err := validateSegment(collection); err != nil {
		return "", appmedia.ErrInvalid
	}
	return path.Join(keyPrefix, modelType, modelID, collection, uniqueFileName(name)), nil
}

// validateSegment rejects empty, dot-relative, and separator-carrying values.
func validateSegment(s string) error {
	if s == "" {
		return appmedia.ErrInvalid
	}
	if s == "." || s == ".." || strings.Contains(s, "/") || strings.Contains(s, "\\") || strings.Contains(s, "\x00") {
		return appmedia.ErrInvalid
	}
	return nil
}

// uniqueFileName preserves the original extension but randomizes the stem so
// uploads never collide. It strips path separators (both slashes) and control
// characters, and caps the length so the resulting key cannot overflow the
// file_name column nor escape the media root.
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
	safe := strings.Map(func(r rune) rune {
		switch {
		case r == '/' || r == '\\' || r == '\x00':
			return '-'
		case r < 0x20:
			return -1
		default:
			return r
		}
	}, strings.TrimSpace(stem))
	safe = strings.TrimLeft(safe, ".")
	if safe == "" {
		safe = "file"
	}
	const maxStem = 128
	if len(safe) > maxStem {
		safe = safe[:maxStem]
	}
	return safe + "-" + hex.EncodeToString(token) + ext
}
