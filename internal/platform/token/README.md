# `internal/platform/token/`

**Technical infrastructure** — implements the `internal/application/token`
contract with HS256 JWTs.

## Key types

| Symbol    | Purpose                                                    |
|-----------|------------------------------------------------------------|
| `Manager` | issues and parses signed access tokens (HS256)             |

## Constructors

- `NewManager(secret, issuer, audience string) *Manager` — builds a JWT
  manager. `issuer` and `audience` are validated on parse; pass empty strings to
  disable the checks.

## Behavior

- `IssueAccessToken` mints an HS256 token carrying the user ID, roles and a `jti`
  (generated when `Claims.JTI` is empty) valid for the given TTL. The token also
  embeds issuer/audience so tokens are tied to the environment they were minted
  for.
- `ParseAccessToken` verifies the HS256 signature, expiry (required, with 30s
  leeway), and issuer/audience (when configured). Any failure returns
  `ErrInvalid` (possibly wrapped).

## Usage

```go
mgr := token.NewManager(cfg.Auth.JWTSecret, cfg.Auth.JWTIssuer, cfg.Auth.JWTAudience)

raw, err := mgr.IssueAccessToken(ctx, apptoken.Claims{UserID: u.ID, Roles: u.Roles}, cfg.Auth.AccessTokenTTL)
claims, err := mgr.ParseAccessToken(ctx, raw)
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
