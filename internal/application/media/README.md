# `internal/application/media/`

**Cross-cutting capability** — the media-library contract (Laravel
media-library style): files attached to any model. Business modules depend on
this interface only; the concrete implementation (database + storage driver)
lives in `internal/platform/media`.

## Key types

| Symbol          | Purpose                                                    |
|-----------------|------------------------------------------------------------|
| `Library`       | programmatic surface: `AddMedia`, `GetMedia`, `ListByModel`, `RemoveMedia`, `URL` |
| `Media`         | one attached file: metadata row plus the storage key of the actual object |
| `AddMediaInput` | what is needed to attach a file: model/collection, display name, MIME type, size, `io.Reader` |

## Errors

`ErrNotFound`, `ErrInvalid` (both map to `apierr` kinds) and `ErrNoURL`
(returned by `URL` when the backing storage cannot expose a public URL, e.g. the
local filesystem).

Well-known collections: `CollectionDefault`, `CollectionAvatar`.

## Usage

```go
m, err := lib.AddMedia(ctx, media.AddMediaInput{
    ModelType:  "user",
    ModelID:    userID,
    Collection: media.CollectionAvatar,
    Name:       "avatar.png",
    MimeType:   "image/png",
    Size:       size,
    Reader:     file,
})

items, err := lib.ListByModel(ctx, "user", userID, "")
url, err := lib.URL(ctx, m.ID)
```

Implemented by `internal/platform/media` (`Service`), injectable into any module
or adapter. It has no HTTP endpoints; expose it over HTTP with a thin adapter in
the composition root if needed.

## Dependency rules

Vendor-free contract; imports stdlib and `internal/application/apierr` only.
