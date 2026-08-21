# Go Project Starter

> **Status: Under active development — not production ready.**
> APIs, schema, and directory layout may change without notice.

A starter Go project following the **modular monolith** architecture: one
deployable binary, isolated business modules, and explicit dependencies
between modules.

```text
cmd/                    entry points / binaries (api, web, worker, scheduler, subscriber, migrate, seed)
internal/platform/      technical infrastructure (how the system does something)
internal/application/   cross-cutting capabilities (contracts: auth, cache, queue, pubsub, storage, mailer, sms, ...)
internal/modules/       business modules (what the business does)
migrations/             SQL migrations (MySQL & PostgreSQL compatible)
storage/                local storage root for the `local` storage driver
public/                 static assets (logo, CSS, images) served at /assets/*
```

Each layer has a README describing what belongs there — see
`cmd/README.md`, `internal/README.md`, `internal/platform/README.md`,
`internal/application/README.md`, `internal/modules/README.md`,
`migrations/README.md`, and `storage/README.md`.

## Architecture

The codebase follows a **layered modular monolith** with strict dependency
rules enforced by `go-arch-lint` (`arch.yaml`) and a grep check in
`make check`.

### Layers (inside-out)

```text
┌─────────────────────────────────────────────────────────────┐
│  cmd/                  composition root (wires everything)  │
├─────────────────────────────────────────────────────────────┤
│  adapter/              HTTP, CLI, schedule, pubsub handlers │
├─────────────────────────────────────────────────────────────┤
│  application/          use cases (commands + queries)       │
├─────────────────────────────────────────────────────────────┤
│  domain/               entities, repository interfaces      │
├─────────────────────────────────────────────────────────────┤
│  infrastructure/       SQL, Redis, external SDK impls      │
└─────────────────────────────────────────────────────────────┘

Cross-cutting contracts live in internal/application/ (auth, cache,
queue, pubsub, storage, media, mailer, sms, ...).
Technical drivers live in internal/platform/ (database, cache, queue,
pubsub, storage, mailer, sms, http, clock, token, hash, logger, ...).
```

### Dependency rules

| Layer | May import |
|-------|-----------|
| `domain` | only `application/apierr` (sentinel errors) |
| `application` (use cases) | `domain` + `application/*` contracts + `platform/clock` |
| `infrastructure` | `domain` + `application/*` + `platform/*` |
| `adapter` | its own `application` + `application/*` + `platform/http` |
| `cmd` (composition root) | everything — it wires modules, platform, and adapters |

**Cross-module rule:** modules may only import each other through their
package root (`module.API` / `module.Service`), never through internal
sub-packages (`domain`, `application`, `infrastructure`, `adapter`). This is
enforced by `make check`.

### Composition root

Each binary under `cmd/` is a thin main that:
1. Loads config (`config.Load()`)
2. Opens infrastructure (DB, Redis, cache, queue, pubsub, storage, mailer, SMS)
3. Instantiates modules with their dependencies
4. Mounts HTTP routes (chi router) and starts the server/worker/scheduler

No business logic lives in `cmd/` — it only delegates.

### CQRS

The application layer follows **CQRS** (Command Query Responsibility
Segregation): every state change is a **command**, every read is a **query**.
Both live under `internal/modules/*/application/` in separate packages.
The data layer enforces this separation through split repository interfaces
and optional read-replica support.

```text
internal/modules/{module}/
  domain/              entities, repository interfaces (Read + Write)
  application/
    command/           write-side use cases (one file per command)
    query/             read-side use cases (one file per query)
    port/              outbound port interfaces for cross-module deps
  infrastructure/      repository implementations (SQL, cache, ...)
  adapter/             HTTP, schedule, pubsub, queue handlers
```

#### Repository separation (CQRS at the data layer)

Every domain repository is split into **read** and **write** interfaces:

| Entity | Read Interface | Write Interface |
|--------|---------------|-----------------|
| User | `UserReadRepository` | `UserWriteRepository` |
| RefreshToken | `RefreshTokenReadRepository` | `RefreshTokenWriteRepository` |
| VerificationCode | `VerificationCodeReadRepository` | `VerificationCodeWriteRepository` |
| PendingContactChange | `PendingContactChangeReadRepository` | `PendingContactChangeWriteRepository` |
| Role | `RoleReadRepository` | `RoleWriteRepository` |
| Permission | `PermissionReadRepository` | `PermissionWriteRepository` |
| UserAccess | `UserAccessReadRepository` | `UserAccessWriteRepository` |

Commands inject **write** interfaces; queries inject **read** interfaces.
Infrastructure implementations satisfy both, but the dependency direction
is explicit.

#### Read replica support (optional)

Point `DB_READ_HOST` (and friends) to a MySQL/PostgreSQL read replica.
When set, all query-side reads route to the replica; writes always go to
the primary. When unset, reads fall back to the primary — zero config
change for existing users.

```env
# primary (writes + reads when no replica is configured)
DB_HOST=localhost
DB_PORT=3306

# optional read replica (queries only)
DB_READ_HOST=read-replica.example.com
DB_READ_PORT=3306
```

#### Command (write side)

Each command is a file with three elements:

1. **Command struct** — plain data carrying the input
2. **Result struct** — what the command returns (optional)
3. **Use-case struct** — holds injected deps, exposes `Execute`

```go
// command/register.go

type RegisterCommand struct {
    Name, Email, Phone, Password string
}

type RegisterResult struct {
    UserID       string
    DevEmailCode string
}

type Register struct {
    users  domain.UserWriteRepository // write-only interface
    hasher hash.PasswordHasher
    // ... injected dependencies
}

func (uc *Register) Execute(ctx context.Context, cmd RegisterCommand) (*RegisterResult, error) {
    // validation, business logic, persistence, audit
}
```

Commands that return no meaningful data return only `error`:

```go
func (uc *Logout) Execute(ctx context.Context, cmd LogoutCommand) error { ... }
```

#### Query (read side)

Queries take **primitive parameters** instead of a command struct, and the
receiver is `q` instead of `uc`:

```go
// query/profile.go

type Profile struct {
    users domain.UserReadRepository // read-only interface
    roles port.Roles
}

func (q *Profile) Execute(ctx context.Context, userID string) (*ProfileResult, error) {
    user, err := q.users.FindByID(ctx, userID)
    // ...
}
```

#### Convention summary

| Aspect | Command | Query |
|--------|---------|-------|
| Package | `application/command/` | `application/query/` |
| Input | named struct (`RegisterCommand`) | primitive params (`userID string`) |
| Receiver | `uc` | `q` |
| Method | `Execute(ctx, cmd) → (*Result, error)` | `Execute(ctx, params...) → (*Result, error)` |
| Constructor | `NewRegister(deps...) *Register` | `NewProfile(deps...) *Profile` |
| Repo inject | write interface | read interface |

#### Adapter layer

Handlers in `adapter/api/` translate HTTP → command/query → HTTP:

```go
func (h *handler) register(w http.ResponseWriter, r *http.Request) {
    var req registerRequest
    if err := decodeJSON(w, r, &req); err != nil { return }

    res, err := h.deps.Register.Execute(r.Context(), command.RegisterCommand{
        Name: req.Name, Email: req.Email,
    })
    if err != nil { writeError(w, err); return }
    writeSuccess(w, http.StatusCreated, res)
}
```

Every adapter receives a `Deps` struct bundling all its use-case pointers
by reference — no business logic, no SQL, just request → use-case → response.

#### Wiring

`module.go` constructs repositories (with `readDB` and `writeDB` pools),
builds every command/query, and assembles them into the public `API`
struct. The composition root (`cmd/api/main.go`) opens both DB pools and
calls `module.New(deps)` then `module.RegisterAPI(router)`.

## Binaries (`cmd/`)

| Command       | Purpose                                                              |
| ------------- | -------------------------------------------------------------------- |
| `api`         | HTTP API server (`APP_PORT`, default 32100) — auth, media, RBAC routes + `/assets/*` |
| `web`         | Public web server (`WEB_PORT`, default 32101) — homepage + `/assets/*` |
| `worker`      | Queue worker — processes email/SMS tasks enqueued by the API         |
| `scheduler`   | Periodic-job runner (`cmd/scheduler`) — cron jobs registered by modules |
| `subscriber`  | Pub/sub consumer (`cmd/subscriber`) — topic handlers against the configured broker |
| `migrate`     | Migration CLI — applies/reverts `migrations/` (`up`, `down`, `version`) |
| `seed`        | One-off seeder (like `artisan db:seed`) — creates default roles/     |
|               | permissions and demo users defined in the module seeder packages      |

The API and worker share the same modules; email/SMS are enqueued by the API
and delivered by the worker. The web binary serves the public homepage
independently on its own port.

## Features

- **Auth module** (`internal/modules/auth`)
  - Register (email and/or phone) + email/phone OTP verification
  - Login with email/password (email or phone identifier) + rate limiting
  - **Magic link login** — no password needed: request a one-time link, click it, get tokens
  - Forgot / reset password
  - Access token (JWT, carries roles) + rotating refresh token, logout, `GET /me`
  - **MFA / TOTP** — setup, confirm, disable authenticator; recovery codes for
    backup; `POST /mfa/verify` during login when TOTP is active
  - **Session management** — family-based refresh tokens with max-session cap
    (`AUTH_MAX_SESSIONS`); list sessions, revoke individual sessions
  - **Profile updates** — change name; changing email/phone records a pending
    change + issues a new OTP (applied on verify)
  - **Password change** — separate endpoint (`POST /me/password`) for
    authenticated users
- **RBAC module** (`internal/modules/rbac`) — Spatie-permissions-style
  - Roles and permissions tables (`roles`, `permissions`, `role_permissions`,
    `user_roles`, `user_permissions`)
  - Direct user permissions plus permissions inherited through roles
  - `super_admin` implicitly has every permission; every new user gets the
    `user` role on registration
  - Versioned permission cache (redis/memory) — changes propagate within `RBAC_CACHE_TTL`
  - `platform/http.RequirePermission(authorizer, "rbac.manage")` middleware
  - Admin API to manage roles, permissions, and user assignments
- **Homepage module** (`internal/modules/homepage`) — demo module with all
  four adapter types: API (`GET /` JSON branding), Web (HTML welcome page),
  Schedule (`homepage.tick` cron job → publishes `app.demo.ping`),
  PubSub (subscribes to `app.demo.ping`)
- **Media library** — Laravel media-library-style cross-cutting capability
  - Contract in `internal/application/media` (`media.Library`), implemented in
    `internal/platform/media`
  - Files attached to any model/collection, metadata in the `media` table
  - Callable from any module: `AddMedia`, `GetMedia`, `ListByModel`,
    `RemoveMedia`, `URL` (backs onto `internal/application/storage` for signed
    or direct object URLs)
- **Queue** — backends: `asynq` (Redis, default) or `db` (database table,
  `queue_jobs`). Emails/SMS are enqueued by the API and sent by a separate
  worker (`cmd/worker`).
- **Pub/Sub** — broadcast events across processes. Contract in
  `internal/application/pubsub`, brokers in `internal/platform/pubsub`:
  `memory` (in-process), `redis` (fire-and-forget), `rabbitmq` (durable, ack),
  `kafka` (durable log, consumer-group fan-out). Demo: `cmd/scheduler` publishes
  `app.demo.ping` every minute → `cmd/subscriber` logs it. Unlike the queue,
  every subscriber receives every message and there is no retry.
- **Mailer** — drivers: `log`, `smtp`, `ses` (Amazon SESv2). Supports text,
  HTML and **attachments**.
- **SMS** — drivers: `log`, `twilio`.
- **Storage** — drivers: `local` (files under `./storage`) and `s3`
  (real AWS or S3-compatible: MinIO, R2, Ceph).
- **Public assets** — files in `public/` served at `/assets/*` by the API and
  web servers. Email HTML and web templates reference them via `ASSETS_BASE_URL`
  (defaults to `APP_BASE_URL`, can point at a CDN). Put `logo.png` etc. in
  `public/assets/`.
- **Cache** — drivers: `redis`, `memory`, `db` (Laravel-style `cache` table).
- **Audit logging** — immutable `audit_logs` table; every write is recorded
  with actor type (`user`/`system`), action (`created`/`updated`/`deleted`),
  and optional old/new JSON diffs. `platform/http.AuditActor` middleware
  injects the actor; `platform/audit.RecordBestEffort()` writes best-effort.
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

The fastest path is the included Docker Compose stack (MySQL, Redis, and the
RabbitMQ/Kafka brokers behind a profile), which builds and runs every binary
for you:

```sh
docker compose up -d            # mysql + redis + api + web + worker + scheduler + subscriber
docker compose --profile brokers up -d   # + rabbitmq + kafka
```

This applies migrations and seeds automatically, then serves the API at
`http://localhost:32100` and the web front at `http://localhost:32101`. To
watch the pub/sub demo, follow the logs of the scheduler and subscriber.

Alternatively, run locally against your own MySQL/Redis. Copy `.env.example`
to `.env` — every binary auto-loads it via `config.Load()` (real environment
variables always win over the file, so docker-compose/CI secrets keep
precedence):

```sh
go mod tidy
go run ./cmd/migrate up
go run ./cmd/seed         # create default roles/permissions + demo users
go run ./cmd/api          # http://localhost:32100
go run ./cmd/worker       # separate terminal — processes email/SMS tasks
go run ./cmd/web          # optional — public homepage on http://localhost:32101
go run ./cmd/scheduler    # optional — cron jobs + publishes the pub/sub demo
go run ./cmd/subscriber   # optional — logs pub/sub demo events (needs a broker)
```

The pub/sub demo needs a broker (`PUBSUB_DRIVER=redis|rabbitmq|kafka`) for the
two binaries to talk; `memory` is in-process only. See
`internal/platform/pubsub/README.md`.

In development (`APP_ENV=development`) OTP codes and magic links are returned
in the API responses so you can exercise the flows end-to-end.

## API

### Homepage

| Method | Path | Auth | Description                        |
|--------|------|------|------------------------------------|
| GET    | `/`  | —    | app branding: `app_name`, `base_url`, `assets_base_url`, `year` |

### Auth

| Method | Path                              | Auth  | Description                       |
|--------|-----------------------------------|-------|-----------------------------------|
| POST   | `/api/v1/auth/register`           | —     | register, returns dev OTPs        |
| POST   | `/api/v1/auth/login`              | —     | login (email/phone + password)    |
| POST   | `/api/v1/auth/mfa/verify`         | —     | verify TOTP code during login     |
| POST   | `/api/v1/auth/magic-link`         | —     | request a magic login link        |
| POST   | `/api/v1/auth/magic-link/verify`  | —     | exchange the token for credentials |
| GET    | `/api/v1/auth/magic-link/verify`  | —     | magic link click (GET, for email links) |
| POST   | `/api/v1/auth/verify-email`       | —     | verify email with OTP (applies a pending email change) |
| POST   | `/api/v1/auth/verify-phone`       | —     | verify phone with OTP (applies a pending phone change) |
| POST   | `/api/v1/auth/forgot-password`    | —     | request a reset code              |
| POST   | `/api/v1/auth/reset-password`     | —     | reset password with code          |
| POST   | `/api/v1/auth/refresh`            | —     | rotate refresh token              |
| POST   | `/api/v1/auth/logout`             | Bearer | revoke refresh token              |
| GET    | `/api/v1/auth/me`                 | Bearer | current user profile + roles/permissions |
| PATCH  | `/api/v1/auth/me`                 | Bearer | update name; changing email/phone records a pending change + issues a new OTP (applied on verify) |
| POST   | `/api/v1/auth/me/password`        | Bearer | change password (authenticated)   |
| GET    | `/api/v1/auth/sessions`           | Bearer | list active sessions (refresh token families) |
| DELETE | `/api/v1/auth/sessions/{familyID}`| Bearer | revoke a session/family          |
| POST   | `/api/v1/auth/mfa/setup`          | Bearer | generate TOTP secret + provisioning URI |
| POST   | `/api/v1/auth/mfa/confirm`        | Bearer | confirm TOTP setup with a valid code |
| POST   | `/api/v1/auth/mfa/disable`        | Bearer | disable TOTP (requires current password) |

### RBAC (admin, requires `rbac.manage`)

| Method | Path                                      | Description                |
|--------|-------------------------------------------|----------------------------|
| GET    | `/api/v1/rbac/roles`                      | list roles                 |
| POST   | `/api/v1/rbac/roles`                      | create role                |
| GET    | `/api/v1/rbac/roles/{code}`               | get role by code           |
| PUT    | `/api/v1/rbac/roles/{code}`               | update role                |
| DELETE | `/api/v1/rbac/roles/{code}`               | delete role                |
| PUT    | `/api/v1/rbac/roles/{code}/permissions`   | sync a role's permissions  |
| GET    | `/api/v1/rbac/permissions`                | list permissions           |
| POST   | `/api/v1/rbac/permissions`                | create permission          |
| PUT    | `/api/v1/rbac/permissions/{code}`         | update permission          |
| DELETE | `/api/v1/rbac/permissions/{code}`         | delete permission          |
| GET    | `/api/v1/rbac/users/{userID}`             | user's roles/permissions   |
| POST   | `/api/v1/rbac/users/{userID}/roles`       | assign role to user        |
| DELETE | `/api/v1/rbac/users/{userID}/roles`       | revoke role from user      |
| POST   | `/api/v1/rbac/users/{userID}/permissions` | grant direct permission    |
| DELETE | `/api/v1/rbac/users/{userID}/permissions` | revoke direct permission   |

> The media library is a programmatic capability (see Features) — it has no
> HTTP endpoints. Modules call `media.Library` directly; if you need to expose
> it over HTTP, add a thin adapter in the composition root.

To grant a user access to protected endpoints, use the RBAC API, e.g. assign
the `super_admin` role. The seeders create a default admin account
(`admin@example.com` / `password123`) and demo user (`user@example.com` /
`password123`) — see `auth/seeder.DefaultUsers`.

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
- `PUBLIC_DIR` — directory served statically at `/assets/*` (default `./public`).
- `ASSETS_BASE_URL` — absolute base URL for static assets used in email/web
  (empty = `APP_BASE_URL`, so it can point at a CDN).
- `DB_DRIVER=mysql|postgres`
- `DB_READ_HOST` — optional read replica host; when set, queries route here
- `DB_READ_PORT`, `DB_READ_USER`, `DB_READ_PASSWORD`, `DB_READ_NAME`, `DB_READ_SSL_MODE`
- `CACHE_DRIVER=redis|memory|db`
- `STORAGE_DRIVER=local|s3`
- `MAIL_DRIVER=log|smtp|ses`
- `SMS_DRIVER=log|twilio`
- `QUEUE_DRIVER=asynq|db`
- `PUBSUB_DRIVER=memory|redis|rabbitmq|kafka`
- `RBAC_CACHE_TTL`
