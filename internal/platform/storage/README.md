# `internal/platform/storage/`

**Technical infrastructure** — `storage.Storage` driver implementations and the
factory that picks one from `STORAGE_DRIVER`. Each driver lives in its own
subpackage:

- `local/` — `Local` (filesystem-backed, rooted at `STORAGE_LOCAL_DIR`, default
  `./storage`). Rejects keys escaping the root; `URL` returns `storage.ErrNoURL`
  because it has no HTTP server of its own.
- `s3/` — `S3` (AWS S3 or S3-compatible: MinIO, Cloudflare R2, Ceph via AWS SDK
  v2). Empty endpoint means real AWS; `UsePathStyle` enables MinIO-style
  services. Implements `storage.Presigner` and `storage.URLGenerator` (signed
  URLs), so private objects can be served without a public bucket.

## Factory

`New(cfg config.StorageConfig) (storage.Storage, error)` — returns the driver for
`cfg.Driver` (`""` is treated as `local`); unknown drivers return an error.

## Usage

```go
store, err := storage.New(cfg.Storage)
if err != nil { return err }

if err := store.Put(ctx, "media/...", reader); err != nil { ... }
if p, ok := store.(storage.Presigner); ok {
    url, err := p.Presign(ctx, key)
}
```

Every S3 operation runs under a 30s timeout so a stuck endpoint cannot hang a
worker; missing keys map to `storage.ErrNotFound`.

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
