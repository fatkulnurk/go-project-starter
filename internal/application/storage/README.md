# `internal/application/storage/`

**Cross-cutting capability** — the object-storage contract. Business modules
only know `Storage`/`Presigner`/`URLGenerator`; the concrete driver (`local`,
`s3`, ...) is chosen in the composition root.

## Key types

| Symbol          | Purpose                                                    |
|-----------------|------------------------------------------------------------|
| `Storage`       | key-based object store: `Put`, `Get`, `Delete`, `Attributes` |
| `Presigner`     | optional; drivers that can produce time-limited signed URLs  |
| `URLGenerator`  | optional; drivers that can produce a public URL for an object |
| `ObjectAttrs`   | metadata about a stored object (currently `Size`)           |

## Errors

`ErrNotFound` (missing object), `ErrInvalidKey` (empty or path traversal),
`ErrNoURL` (driver cannot produce a public URL, e.g. local filesystem).

## Usage

```go
if err := store.Put(ctx, key, reader); err != nil { return err }

rc, err := store.Get(ctx, key)
defer rc.Close()

if p, ok := store.(storage.Presigner); ok {
    url, err := p.Presign(ctx, key)
}
```

Keys are opaque to callers but must be driver-safe; invalid keys are rejected
with `ErrInvalidKey`. Implemented by `internal/platform/storage` (`local`,
`s3`) behind the `storage.New` factory.

## Dependency rules

Vendor-free contract; imports stdlib only.
