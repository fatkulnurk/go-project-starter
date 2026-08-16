# `internal/application/auth/`

**Cross-cutting capability** — the authentication contract: *who is calling the
system*. Implementations live behind the `Authenticator` interface (JWT, opaque
tokens, sessions), so business modules never depend on a token library.

## Key types

| Symbol           | Purpose                                                    |
|------------------|------------------------------------------------------------|
| `Authenticator`  | resolves a raw credential (e.g. bearer token) to an `*Identity`, or `ErrUnauthenticated` |
| `Identity`       | the authenticated caller: `UserID` + `Roles` asserted when the credential was issued |

## Helpers

- `ErrUnauthenticated` — returned when the credential is missing, invalid,
  revoked, or expired. Maps to the `unauthenticated` API code.
- `WithIdentity(ctx, id)` / `IdentityFrom(ctx)` — stash and recover the identity
  in the context; HTTP middleware uses this so handlers can recover the caller
  without threading it through every signature.

## Usage

```go
id, err := authenticator.Authenticate(r.Context(), raw)
if err != nil {
    // treat as unauthenticated (401)
}
ctx := auth.WithIdentity(r.Context(), id)
```

The `internal/platform/http.Authenticate` middleware does exactly this on
protected routes.

## Dependency rules

Vendor-free contract; imports stdlib only. The concrete implementation
(`internal/platform/token` JWT) is chosen in the composition root.
