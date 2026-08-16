# `internal/modules/auth/`

**Business module** — the authentication module: registration, password and
magic-link login, email/phone OTP verification, forgot/reset password, token and
session management, and TOTP MFA with recovery codes. It depends only on
cross-cutting application contracts and the repositories wired through
`Dependencies`.

## What it does

- Register by email and/or phone + email/phone OTP verification
- Login with password (email or phone identifier), with a TOTP MFA challenge
  when the user enabled it
- Magic-link login — request a one-time link, click it, get tokens
- Forgot / reset password
- Access token (JWT) + rotating refresh token, logout, `GET /me`, profile update
- Sessions (refresh-token families), per-session revoke, max active sessions
- Change password (revokes other sessions), TOTP setup/confirm/disable

## How it works — request flow

```text
HTTP /api/v1/auth/*            (adapter/api)
        │  parse request → one use case → standardized response
        ▼
application/command|query      (use cases, business rules)
        │  domain entities + repositories + application contracts
        ▼
domain/ + infrastructure/      (pure rules + database/sql repositories)
```

Every handler in `adapter/api` only parses the request, calls **one** command or
query from `API`, and renders the response via `platform/http` helpers — no
business logic or SQL in the adapter.

## Layers

| Layer            | Contents                                                              |
|------------------|-----------------------------------------------------------------------|
| `domain/`        | `User`, `VerificationCode`, `RefreshToken`, `PendingContactChange`, `RecoveryCode` + the repository interfaces use cases depend on |
| `application/command/` | one struct per use case (`Register`, `Login`, `VerifyEmail`, `Refresh`, `SetupTOTP`, ...); `NewTokenIssuer` (token + session minting), `NewJTIDenylist`, `NewMFAChallenges`, `NewRateLimiter` |
| `application/query/` | read-only use cases: `Profile`, `Sessions`                     |
| `application/port/` | `Roles` — the narrow RBAC port the module needs (resolved via `rbacAdapter` to `rbac.Service`) |
| `application/task/` | queue task names + JSON payloads + `Enqueue*` helpers           |
| `infrastructure/`  | `database/sql` repositories (users, refresh tokens, verification codes, pending changes, recovery codes) |
| `adapter/api/`     | routes under `/api/v1/auth` — public rate-limited group + authenticated group |
| `adapter/queue/`   | worker task handlers that render templates and send email/SMS   |
| `seeder/`          | `UserSeeder` — demo users (`admin@example.com`, `user@example.com`), registered as `auth.users` |
| `template/`        | module-owned email/SMS copy, split by channel, rendered over the platform layouts |

## Async delivery (queue)

Commands never send mail/SMS directly — they enqueue tasks via the
`application/queue.Enqueuer` contract:

```text
Register/UpdateProfile/ForgotPassword/MagicLinkRequest
   └─► task.Enqueue*("auth:send_verification_email", ...)
        └─► worker (cmd/worker) → adapter/queue handler
              ├─ decode payload
              ├─ ProcessForgotPassword / ProcessMagicLink resolve identifier → user + code/link (inside the worker)
              ├─ template.Email()/SMS() render module copy over the platform layout
              └─ mailer.MailSender / sms.Sender deliver
```

Unknown identifiers are skipped silently (no account-probing leak). Malformed
payloads return `queue.ErrPermanent` so they are never retried.

## Tokens & sessions

- `NewTokenIssuer` mints the access token (`token.Manager`) + a hashed refresh
  token row, records the `jti`, and groups sessions into families.
- `NewJTIDenylist` (cache) denies revoked access tokens (logout, session revoke,
  password change). The denylist **fails open** on cache errors so an
  unavailable cache never locks everyone out.
- `NewMFAChallenges` (cache, 5m TTL) stores the pending TOTP challenge between
  the first and second step of a protected login; `NewRateLimiter` throttles
  login/forgot/magic-link/public endpoints.
- `Authenticator()` exposes the module's authenticator to the shared
  `platform/http.Authenticate` middleware.

## Public surface (`module.go`)

- `New(deps Dependencies) *Module` — composition inside the module: builds
  repositories, rate-limit/denylist/challenge stores, and every use case.
- `API` (`api.go`) — the use cases other code may call; modules depend only on
  this.
- `RegisterAPI(r)` — mounts the HTTP routes; `RegisterQueue(r)` — registers the
  worker handlers; `Authenticator()` — for the shared middleware.

## Cross-module dependency

Auth depends on the RBAC module **only through `rbac.Service`**, behind the
narrow `port.Roles` interface (`rbacAdapter` in `module.go`). It is nil-safe:
without RBAC wired, assigning the default `user` role is skipped.
