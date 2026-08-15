// Package hash implements the application/hash contract.
package hash

import (
	"context"

	apphash "github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"golang.org/x/crypto/bcrypt"
)

// Hash hashes passwords with bcrypt.
type Hash struct {
	cost int
}

var _ apphash.PasswordHasher = (*Hash)(nil)

// NewHash builds a bcrypt hasher (cost 0 = DefaultCost).
func NewHash(cost int) *Hash {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &Hash{cost: cost}
}

// Hash implements apphash.PasswordHasher.
func (h *Hash) Hash(_ context.Context, password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	return string(b), err
}

// Compare implements apphash.PasswordHasher.
func (h *Hash) Compare(_ context.Context, password, storedHash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
}
