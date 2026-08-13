// Package hash defines the cross-cutting password hashing contract.
package hash

import "context"

// PasswordHasher hashes and compares passwords. Implementations (bcrypt,
// argon2, ...) are chosen in the composition root.
type PasswordHasher interface {
	// Hash returns a self-contained string (salt embedded).
	Hash(ctx context.Context, password string) (string, error)
	// Compare reports whether password matches the stored hash.
	Compare(ctx context.Context, password, storedHash string) bool
}
