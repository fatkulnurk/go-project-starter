# `internal/platform/media/`

**Technical infrastructure** — implements the `internal/application/media`
contract: metadata is persisted in the `media` table and file bytes live behind
the application storage driver (`local`, `s3`).

## Key types

| Symbol       | Purpose                                                    |
|--------------|------------------------------------------------------------|
| `Service`    | `appmedia.Library` implementation orchestrating storage + repository |
| `Repository` | persists media records in the `media` table (`database/sql`, `?` placeholders rebound for postgres) |
| `Deps`       | wiring for `New`: `Repo`, `Storage`, optional `URLGenerator`/`Auditor`/`Clock`, `Disk` name |

## Constructors

- `New(deps Deps) *Service` — builds the service; `Repo` and `Storage` are
  required, `URLGenerator`/`Auditor`/`Clock` may be left zero (nil).
- `NewRepository(db *sql.DB, driver string) *Repository`.

## Behavior

- `AddMedia` writes the file under a generated object key, then persists the
  metadata record; on any failure the already-stored object is deleted so no
  orphan is left behind.
- Object keys are `media/{modelType}/{modelID}/{collection}/{base-<rand>}{ext}`,
  built by `ObjectKey(...)`; segments are validated against path traversal.
- `URL` resolves the media's storage key through the configured
  `storage.URLGenerator`; `ErrNoURL` when none is configured (e.g. local disk).
- `AddMedia`/`RemoveMedia` record audit entries when an `Auditor` is wired in.

## Usage

```go
service := media.New(media.Deps{
    Repo:         media.NewRepository(db, cfg.Database.Driver),
    Storage:      store,
    URLGenerator: store.(storage.URLGenerator), // s3 only; nil for local
    Disk:         cfg.Storage.Driver,
    Auditor:      auditRecorder,
    Clock:        clock.Real{Loc: cfg.Location()},
})

m, err := service.AddMedia(ctx, appmedia.AddMediaInput{ ... })
u, err := service.URL(ctx, m.ID)
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
