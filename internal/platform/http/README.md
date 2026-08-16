# `internal/platform/http/`

**Technical infrastructure** — the HTTP server and shared middleware for the
API: chi router, auth/RBAC guards, error mapping, standardized responses,
security headers, CORS, rate limiting and static file serving.

## Server & router

| Symbol            | Purpose                                                    |
|-------------------|------------------------------------------------------------|
| `NewRouter(opts...)` | chi router with the standard chain: RequestID, Recoverer, `LoggerMiddleware`, `AuditActor`, `SecurityHeaders`, `CORS`, 30s timeout. `RouterOptions.CORSAllowedOrigins` enables cross-origin access (empty = same-origin only) |
| `NewServer(port, router)` | `http.Server` with production timeouts (15s read, 5s read-header, 30s write, 60s idle) and 1 MiB header cap |

## Middleware

| Symbol                | Purpose                                                    |
|-----------------------|------------------------------------------------------------|
| `Authenticate(authz)` | resolves a Bearer token via the `auth.Authenticator` contract, stores the identity + audit actor in the context; unauthenticated → 401 |
| `RequirePermission(authz, perm)` | RBAC permission guard (after `Authenticate`); missing identity → 401, denied → 403 |
| `RequireRole(authz, role)` | RBAC role guard (after `Authenticate`); missing identity → 401, denied → 403 |
| `RateLimitByIP(c, max, window)` | per-client-IP throttling via the shared cache; excess → 429 |
| `LoggerMiddleware`   | slog access log (method, path, status, duration, request id); supports SSE/websocket/hijack |
| `SecurityHeaders()`  | `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Content-Security-Policy` |
| `CORS(origins)`      | per-origin allow list; empty disables cross-origin access |
| `AuditActor`         | captures client IP + user agent and stores an `audit.Actor` in the context |
| `SetTrustedProxies(entries)` | configures which IPs/CIDRs may set `X-Forwarded-For` |
| `MountStatic(r, dir, base)` | serves `publicDir` at `basePath` (e.g. `/assets`) |

## Responses

| Symbol                 | Purpose                                                    |
|------------------------|------------------------------------------------------------|
| `WriteSuccess` / `WriteSuccessMessage` | success envelope `{"data": ...}` |
| `WriteError`           | error envelope `{"error": {"code", "message"}}` with explicit status/code |
| `WriteMappedError`     | derives status + code from `apierr` kinds / sentinels; unknown errors → 500 "internal" without leaking the message |
| `DecodeJSON`           | parses the body (max 1 MiB) into `v`; malformed → `apierr.ErrInvalid`, oversized → `apierr.ErrPayloadTooLarge` |
| `BearerToken(r)`       | extracts the token from the `Authorization` header |
| `ClientIP(r)`          | caller IP honoring trusted proxies |

## Usage

```go
r := http.NewRouter(http.RouterOptions{CORSAllowedOrigins: cfg.CORSAllowedOrigins})
http.SetTrustedProxies(cfg.TrustedProxies)

r.Route("/api/v1", func(r chi.Router) {
    r.Post("/auth/login", authModule.Login)
    r.Group(func(r chi.Router) {
        r.Use(http.Authenticate(authenticator))
        r.Use(http.RequirePermission(authorizer, authorization.PermissionManageRBAC))
        r.Get("/rbac/roles", rbacModule.ListRoles)
    })
})
http.MountStatic(r, cfg.PublicDir, "/assets")
srv := http.NewServer(cfg.Port, r)
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
