# `internal/application/hash/`

**Cross-cutting capability** — the password hashing contract. Callers never
depend on a specific algorithm or its cost parameters; the implementation
(bcrypt, argon2, ...) is chosen in the composition root.

## Key types

| Symbol           | Purpose                                                    |
|------------------|------------------------------------------------------------|
| `PasswordHasher` | hashes and compares passwords                               |

## Contract surface

- `Hash(ctx, password) (string, error)` — returns a self-contained string (salt
  embedded), suitable for storage in a single column.
- `Compare(ctx, password, storedHash) bool` — reports whether the password
  matches the stored hash; returns `false` (never an error) for mismatched or
  unparseable hashes.

## Usage

```go
hashed, err := hasher.Hash(ctx, "secret")
// store hashed ...

if hasher.Compare(ctx, "secret", hashed) { /* ok */ }
```

Implemented by `internal/platform/hash` (`BCrypt` via `NewBCrypt`).

## Dependency rules

Vendor-free contract; imports stdlib only.
