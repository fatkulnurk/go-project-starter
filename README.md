# Go Project Starter

A starter Go project following the **modular monolith** architecture: one
deployable binary, isolated business modules, and explicit dependencies
between modules.

```text
cmd/                    entry points (composition roots)
internal/platform/      technical infrastructure (how the system does something)
internal/application/   cross-cutting capabilities (contracts: auth, cache, queue, storage, mailer, sms, ...)
internal/modules/       business modules (what the business does)
migrations/             SQL migrations (MySQL & PostgreSQL compatible)
storage/                local storage root for the `local` storage driver
```

Each layer has a README describing what belongs there — see
`internal/README.md`, `internal/platform/README.md`,
`internal/application/README.md`, and `internal/modules/README.md`.

## Features

- **Auth module** (`internal/modules/auth`)
  - Register (email and/or phone) + email/phone OTP verification
  - Login with email/password (email or phone identifier) + rate limiting
  - **Magic link login** — no password needed: request a one-time link, click it, get tokens
  - Forgot / reset password
  - Access token (JWT, carries roles) + rotating refresh token, logout, `GET /me`
- **RBAC module** (`internal/modules/rbac`) — Spatie-permissions-style
  - Roles and permissions tables (`roles`, `permissions`, `role_permissions`,
    `user_roles`, `user_permissions`)
  - Direct user permissions plus permissions inherited through roles
  - `super_admin` implicitly has every permission; every new user gets the
    `user` role on registration
  - Versioned permission cache (redis/memory) — changes propagate within `RBAC_CACHE_TTL`
  - `platform/http.RequirePermission(authorizer, "rbac.manage")` middleware
  - Admin API to manage roles, permissions, and user assignments
- **Media module** (`internal/modules/media`) — Laravel media-library-style
  - Files attached to any model/collection, metadata in the `media` table
  - Upload / list / metadata / download / delete endpoints
- **Queue** — hibiken/asynq (Redis). Emails/SMS are enqueued by the API and
  sent by a separate worker (`cmd/worker`).
- **Mailer** — drivers: `log`, `smtp`, `ses` (Amazon SESv2). Supports text,
  HTML and **attachments**.
- **SMS** — drivers: `log`, `twilio`.
- **Storage** — drivers: `local` (files under `./storage`) and `s3`
  (real AWS or S3-compatible: MinIO, R2, Ceph).
- **Cache** — drivers: `redis`, `memory`.
- **Database** — `mysql` (default) or `postgres`, driven by `database/sql`
  with portable SQL (`?` placeholders rebound for postgres).
- **Migrations** — golang-migrate format, run via `cmd/migrate`.

## Response conventions

Every JSON endpoint uses one envelope:

```jsonc
// success
{ "data": { ... } }
// error
{ "error": { "code": "not_found", "message": "not found" } }
```

Errors are mapped centrally by `platform/http.WriteMappedError` from shared
sentinels in `internal/application/apierr`. No magic strings/numbers in code —
literals live in constants (see `config`, `permission`, the DTO files).

## Quick start

1. Copy `.env.example` to `.env` and set `DB_*`, `REDIS_ADDR`, and the auth
   secret. Default mail/SMS drivers are `log` so the starter runs with no
   external credentials.
2. Start MySQL and Redis, then:

   ```sh
   go mod tidy
   go run ./cmd/migrate up
   go run ./cmd/api
   go run ./cmd/worker   # separate terminal — processes email/SMS tasks
   ```

3. In development (`APP_ENV=development`) OTP codes and magic links are
   returned in the API responses so you can exercise the flows end-to-end.

## API

### Auth

| Method | Path                      | Auth  | Description                       |
|--------|---------------------------|-------|-----------------------------------|
| POST   | `/api/v1/auth/register`   | —     | register, returns dev OTPs        |
| POST   | `/api/v1/auth/login`      | —     | login (email/phone + password)    |
| POST   | `/api/v1/auth/magic-link` | —     | request a magic login link        |
| POST   | `/api/v1/auth/magic-link/verify` | — | exchange the token for credentials |
| POST   | `/api/v1/auth/verify-email`  | —  | verify email with OTP (applies a pending email change) |
| POST   | `/api/v1/auth/verify-phone`  | —  | verify phone with OTP (applies a pending phone change) |
| POST   | `/api/v1/auth/forgot-password` | — | request a reset code             |
| POST   | `/api/v1/auth/reset-password`  | — | reset password with code         |
| POST   | `/api/v1/auth/refresh`     | —     | rotate refresh token              |
| POST   | `/api/v1/auth/logout`      | Bearer | revoke refresh token            |
| GET    | `/api/v1/auth/me`          | Bearer | current user profile + roles/permissions |
| PATCH  | `/api/v1/auth/me`          | Bearer | update name; changing email/phone records a pending change + issues a new OTP (applied on verify) |

### RBAC (admin, requires `rbac.manage`)

| Method | Path                                   | Description                |
|--------|----------------------------------------|----------------------------|
| GET    | `/api/v1/rbac/roles`                   | list roles                 |
| POST   | `/api/v1/rbac/roles`                   | create role                |
| GET    | `/api/v1/rbac/permissions`             | list permissions           |
| POST   | `/api/v1/rbac/permissions`             | create permission          |
| PUT    | `/api/v1/rbac/roles/{name}/permissions`| replace a role's permissions |
| GET    | `/api/v1/rbac/users/{userID}`          | user's roles/permissions   |
| POST   | `/api/v1/rbac/users/{userID}/roles`    | assign role to user        |
| DELETE | `/api/v1/rbac/users/{userID}/roles`    | revoke role from user      |
| POST   | `/api/v1/rbac/users/{userID}/permissions` | grant direct permission  |
| DELETE | `/api/v1/rbac/users/{userID}/permissions` | revoke direct permission|

### Media

| Method | Path                            | Auth           | Description                  |
|--------|---------------------------------|----------------|------------------------------|
| POST   | `/api/v1/media`                 | Bearer + `media.manage` | multipart upload (`file`, `model_type`, `model_id`, `collection`) |
| GET    | `/api/v1/media?model_type=&model_id=&collection=` | Bearer | list media for a model |
| GET    | `/api/v1/media/{id}`            | Bearer         | media metadata               |
| GET    | `/api/v1/media/{id}/download`   | Bearer         | file bytes (streamed)        |
| DELETE | `/api/v1/media/{id}`            | Bearer + `media.manage` | delete media      |

To grant a user access to protected endpoints, use the RBAC API, e.g. assign
the `super_admin` role, or set `RBAC_BOOTSTRAP_SUPER_ADMIN_EMAIL` before
startup.

## Verification

```sh
make check        # build + vet + test + gofmt + dependency-direction grep
make lint         # golangci-lint run ./...
```

## Configuration

Everything is configured via environment variables — see `.env.example` for
all keys. To swap a driver, change the corresponding `*_DRIVER` variable; the
module code stays unchanged.

- `APP_TIMEZONE` — IANA location (e.g. `Asia/Jakarta`), default `UTC`. The
  whole app follows it: DB session, clock and Go-written timestamps.
- `DB_DRIVER=mysql|postgres`
- `CACHE_DRIVER=redis|memory|db`
- `STORAGE_DRIVER=local|s3`
- `MAIL_DRIVER=log|smtp|ses`
- `SMS_DRIVER=log|twilio`
- `QUEUE_DRIVER=asynq|db`
- `RBAC_CACHE_TTL`, `RBAC_BOOTSTRAP_SUPER_ADMIN_EMAIL`