# `internal/application/cache/`

**Cross-cutting capability** — the cache contract. Business modules depend on
this interface only; the concrete driver (`redis`, `memory`, `db`) is chosen in
the composition root.

## Key types

| Symbol   | Purpose                                                    |
|----------|------------------------------------------------------------|
| `Cache`  | simple key-value store with TTL; values are opaque `[]byte` |
| `ErrNotFound` | returned by `Get`/`GetDelete` when the key is missing or expired |

## Contract surface

`Get`, `Set` (0 TTL = no expiry), `Delete`, `GetDelete` (atomic read-and-remove
for single-use tokens), `Increment` (atomic counter, creates at `delta`),
`Expire` (non-positive TTL deletes), `Ping`, `Close`.

A found value from `Get` is always non-nil, so callers can distinguish a cache
miss from a stored empty payload.

## Usage

```go
if v, err := c.Get(ctx, key); err != nil {
    if errors.Is(err, cache.ErrNotFound) {
        // cache miss
    }
} else {
    // hit
}
```

Implemented by `internal/platform/cache` (`NewRedis`, `NewMemory`,
`NewDatabase`) behind the `cache.New` factory.

## Dependency rules

Vendor-free contract; imports stdlib only.
