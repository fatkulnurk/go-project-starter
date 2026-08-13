// Package hash implements the application/hash contract.
package hash

import (
	"context"

	apphash "github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"golang.org/x/crypto/bcrypt"
)

// Hasher hashes passwords with bcrypt.
type Hasher struct {
	cost int
}

var _ apphash.PasswordHasher = (*Hasher)(nil)

// NewHasher builds a bcrypt hasher (cost 0 = DefaultCost).
func NewHasher(cost int) *Hasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &Hasher{cost: cost}
}

// Hash implements apphash.PasswordHasher.
func (h *Hasher) Hash(_ context.Context, password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	return string(b), err
}

// Compare implements apphash.PasswordHasher.
func (h *Hasher) Compare(_ context.Context, password, storedHash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
}
