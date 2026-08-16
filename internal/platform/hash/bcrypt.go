// Package hash implements the application/hash contract.
package hash

import (
	"context"

	apphash "github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"golang.org/x/crypto/bcrypt"
)

// BCrypt hashes passwords with bcrypt.
// The cost factor is fixed at construction time; higher costs are slower but
// more resistant to brute-force attacks.
type BCrypt struct {
	cost int
}

var _ apphash.PasswordHasher = (*BCrypt)(nil)

// NewBCrypt builds a bcrypt hasher. A cost of 0 selects bcrypt.DefaultCost;
// other values are passed through to bcrypt, which rejects out-of-range costs
// at Hash time.
func NewBCrypt(cost int) *BCrypt {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &BCrypt{cost: cost}
}

// Hash implements apphash.PasswordHasher.
// The context is ignored. It returns the bcrypt-encoded hash, or an error
// when the configured cost is out of range.
func (h *BCrypt) Hash(_ context.Context, password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	return string(b), err
}

// Compare implements apphash.PasswordHasher.
// The context is ignored. It reports whether password matches storedHash; a
// malformed hash simply compares false.
func (h *BCrypt) Compare(_ context.Context, password, storedHash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
}
