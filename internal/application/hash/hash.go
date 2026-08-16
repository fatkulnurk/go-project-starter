// Package hash defines the cross-cutting password hashing contract.
package hash

import "context"

// PasswordHasher hashes and compares passwords. Implementations (bcrypt,
// argon2, ...) are chosen in the composition root, so callers never depend on
// a specific algorithm or its cost parameters.
type PasswordHasher interface {
	// Hash returns a self-contained string (salt embedded). The result is
	// suitable for storage in a single column and for later Compare.
	Hash(ctx context.Context, password string) (string, error)

	// Compare reports whether password matches the stored hash. It returns
	// false (never an error) for mismatched or unparseable hashes.
	Compare(ctx context.Context, password, storedHash string) bool
}
