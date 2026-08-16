# `internal/application/apierr/`

**Cross-cutting capability** — the shared API error sentinels. Every module maps
its errors onto these values (or implements `HTTPStatus()`/`ErrorCode()`) so the
HTTP layer renders one consistent error envelope:

```json
{ "error": { "code": "not_found", "message": "not found" } }
```

It has no dependencies besides stdlib and the cross-cutting application
contracts, so domain packages may safely reference the sentinels.

## Key types

| Symbol          | Purpose                                                        |
|-----------------|----------------------------------------------------------------|
| `Kind`          | pairs an HTTP status with a stable machine-readable code        |
| `Error`         | sentinel error carrying a `Kind`; implements `error`/`StatusCoder`/`ErrorCoder` |
| `StatusCoder`   | interface for errors that know their HTTP status               |
| `ErrorCoder`    | interface for errors that know their machine-readable code     |

## API kinds

`KindUnauthenticated` (401), `KindUnauthorized` (401), `KindForbidden` (403),
`KindNotFound` (404), `KindConflict` (409), `KindVerificationNeeded` (403),
`KindCodeExpired` (410), `KindTooManyRequests` (429), `KindPayloadTooLarge` (413),
`KindInvalid` (422), `KindInternal` (500).

`New(kind, msg)` builds an `Error`, `Wrap(kind, fmt, ...)` builds one with
`fmt.Sprintf` context, and `KindOf(err)` extracts the kind from any error chain
(`KindInternal` when unknown).

## Usage

```go
return nil, apierr.New(apierr.KindNotFound, "not found")
// or, with context:
return nil, apierr.Wrap(apierr.KindInvalid, "invalid phone %q", phone)
```

The HTTP layer maps the result centrally:

```go
http.WriteMappedError(w, err) // picks status + code from the error's kind
```

`Err*` sentinels (e.g. `ErrNotFound`) are usable directly by use cases and
domain packages; `errors.Is` matches them through wrapped errors via `Unwrap`.

## Dependency rules

May import only stdlib and other `internal/application` contracts. No SQL, no
HTTP handlers, no vendor imports.
