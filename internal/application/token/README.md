# `internal/application/token/`

**Cross-cutting capability** — the token contract. Business modules only know
`Manager`; the concrete implementation (JWT, opaque, ...) is chosen in the
composition root.

## Key types

| Symbol    | Purpose                                                    |
|-----------|------------------------------------------------------------|
| `Manager` | issues and parses signed access tokens                      |
| `Claims`  | what is encoded inside an access token: `UserID`, `Roles`, `JTI` |

`JTI` is the token id minted at issue time. The auth module stores it on the
refresh-token row so revoking a session can deny outstanding access tokens
immediately via the denylist.

## Contract surface

- `IssueAccessToken(ctx, c, ttl) (string, error)` — mints an access token for
  the identity; the returned string is opaque to the caller.
- `ParseAccessToken(ctx, raw) (*Claims, error)` — validates `raw` and returns
  its claims; rejects malformed, expired, or differently-signed tokens.

## Usage

```go
raw, err := mgr.IssueAccessToken(ctx, token.Claims{UserID: u.ID, Roles: u.Roles}, cfg.Auth.AccessTokenTTL)

claims, err := mgr.ParseAccessToken(ctx, raw)
```

Implemented by `internal/platform/token` (`Manager`, HS256 JWT via
`NewManager`).

## Dependency rules

Vendor-free contract; imports stdlib only.
