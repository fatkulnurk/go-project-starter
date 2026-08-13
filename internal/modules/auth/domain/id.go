// Package domain holds the pure domain of the auth module: entities, value
// objects and repository interfaces. It must never import frameworks.
package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// newID returns a 64-char hex identifier (32 random bytes).
func newID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("domain: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// NewID exposes the internal ID generator for infrastructure and adapters.
func NewID() string { return newID() }

// NewOpaqueToken returns a cryptographically random 64-char hex token used for
// refresh tokens and magic links.
func NewOpaqueToken() string { return newID() }

// HashSecret returns the hex SHA-256 of raw. Used to store tokens/codes so a
// database leak does not expose usable credentials.
func HashSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
