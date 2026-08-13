package domain

import (
	"crypto/rand"
	"encoding/hex"
	"path"
	"path/filepath"
	"strings"
)

// Storage key prefix for media objects.
const (
	keyPrefix     = "media"
	pathSeparator = "/"
	randTokenSize = 8
)

func newID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("media: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// ObjectKey builds the storage key for a file: media/{modelType}/{modelID}/{collection}/{base-<rand>}{ext}.
func ObjectKey(modelType, modelID, collection, name string) string {
	if collection == "" {
		collection = CollectionDefault
	}
	return path.Join(keyPrefix, modelType, modelID, collection, uniqueFileName(name))
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
