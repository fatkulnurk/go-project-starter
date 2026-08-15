# `internal/platform/`

**Technical infrastructure** — the *how* of the system. Every package here
implements something behind a contract from `internal/application` (or is a
framework-bound helper used by module adapters).

The business modules never import these packages directly except in their
`adapter/` and `infrastructure/` layers. The composition roots (`cmd/`) pick a
driver per environment; modules only ever see the interface.

## What lives here

| Package             | Provides                                                          |
|---------------------|-------------------------------------------------------------------|
| `hash`              | `hash.PasswordHasher` implementation (`bcrypt.go`)                |
| `cache`             | `cache.Cache` implementations: `redis`, `memory` + factory         |
| `clock`             | `Clock` contract implementation (`Real`, `Fixed`)                  |
| `config`            | environment-driven configuration, loaded once at startup; driver-name constants (`DriverMySQL`, `DriverPostgres`, `DriverRedis`, ...) |
| `database`          | `database/sql` pool, DSN, golang-migrate URL, `?`→`$n` rebinding   |
| `http`              | chi router + middleware (auth, errors, standardized responses)     |
| `token`             | `token.Manager` implementation (`jwt.go`, HS256 JWT)               |
| `logger`            | structured logging (slog) setup                                    |
| `mailer`            | `mailer.MailSender` implementations: `log`, `smtp` (pooled), `ses` + MIME; shared email layout (`NewEmailLayout`) |
| `queue`             | asynq client/server glue behind `queue.Enqueuer`                   |
| `sms`               | `sms.Sender` implementations: `log`, `twilio`                      |
| `storage`           | `storage.Storage` implementations: `local`, `s3` + factory         |
| `audit`             | `audit.Recorder` implementation: `SQLRecorder` (writes `audit_logs`)|
| `view`              | shared base HTML layout (`NewLayout`) for browser pages            |

## What belongs here

- One driver per third-party concern, wrapped behind an application contract.
- **Factories** (`New(cfg)`) that return the contract interface.
- Framework-specific glue that modules shouldn't touch directly.
- Each package may import other `platform` packages and `internal/application`,
  but never `internal/modules`.

## What does NOT belong here

- Business logic (authentication flows, role checks, media metadata rules).
- Use cases with domain entities — those live in `internal/modules`.
- Anything that only one business module uses; start inside the module and only
  promote it to `platform` when a second consumer appears.

Rule of thumb: `platform` owns **frameworks and drivers**; if you are wrapping
a library, this is home. If you are describing what the business wants, that's
a contract in `internal/application` and lives in a module.

### Connection pools

All outbound clients use configurable pools instead of per-call connections:

- `database` — pool size/lifetime from `DB_MAX_OPEN_CONNS`,
  `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`.
- `cache` (redis) — `REDIS_POOL_SIZE`, `REDIS_MIN_IDLE_CONNS`.
- `queue` (asynq) — `QUEUE_REDIS_POOL_SIZE`. The queue reuses `REDIS_ADDR` /
  `REDIS_PASSWORD` but runs in its **own logical Redis DB** (`QUEUE_REDIS_DB`,
  default `1`) so cache keys (`REDIS_DB`, default `0`) and queue keys never
  collide on a shared Redis server.
- `mailer` (smtp) — `MAIL_SMTP_POOL_SIZE`; a bounded `net/smtp` connection
  pool modeled after `knadh/smtppool` reuses authenticated connections,
  retries retriable (network) errors, sweeps idle connections, and drops
  failed connections. TLS is controlled by `MAIL_SMTP_SSL`: `none`, `tls`
  (implicit TLS, e.g. port 465), or `starttls` (default).
- `ses` and `s3` rely on the AWS SDK's built-in HTTP connection pool.

### Shared layouts

Two shared HTML layouts live in `platform` so every module renders consistently:

- `mailer.NewEmailLayout()` — the email shell (brand header, body block,
  footer). Auth attaches its email content templates via `ParseFS`.
- `view.NewLayout()` — the browser page shell used by web pages (e.g. the
  welcome page at `/`).

Both define a `layout` template with a `content` block; modules attach their
own content definitions with `(*html/template.Template).ParseFS`.
