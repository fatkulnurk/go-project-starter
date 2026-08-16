# `internal/platform/hash/`

**Technical infrastructure** — implements the `internal/application/hash`
contract.

## Key types

| Symbol | Purpose                                                    |
|--------|------------------------------------------------------------|
| `BCrypt`| `hash.PasswordHasher` implementation using bcrypt           |

## Constructors

- `NewBCrypt(cost int) *BCrypt` — builds a bcrypt hasher. A cost of `0` selects
  `bcrypt.DefaultCost`; other values are passed through to bcrypt (out-of-range
  costs are rejected at `Hash` time).

## Behavior

- `Hash(ctx, password)` — returns the bcrypt-encoded hash (self-contained salt);
  the context is ignored.
- `Compare(ctx, password, storedHash)` — reports whether the password matches; a
  malformed hash simply compares `false`.

## Usage

```go
hasher := hash.NewBCrypt(0) // default cost

hashed, err := hasher.Hash(ctx, "secret")
ok := hasher.Compare(ctx, "secret", hashed)
```

## Dependency rules

May import `internal/application`; never `internal/modules`.
