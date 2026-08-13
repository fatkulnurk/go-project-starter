# `internal/platform/`

**Technical infrastructure** — the *how* of the system. Every package here
implements something behind a contract from `internal/application` (or is a
framework-bound helper used by module adapters).

The business modules never import these packages directly except in their
`adapters/` and `infrastructure/` layers. The composition roots (`cmd/`) pick a
driver per environment; modules only ever see the interface.

## What lives here

| Package             | Provides                                                          |
|---------------------|-------------------------------------------------------------------|
| `hash`              | `hash.PasswordHasher` implementation (`bcrypt.go`)                |
| `cache`             | `cache.Cache` implementations: `redis`, `memory` + factory         |
| `clock`             | `Clock` contract implementation (`Real`, `Fixed`)                  |
| `config`            | environment-driven configuration, loaded once at startup           |
| `database`          | `database/sql` pool, DSN, golang-migrate URL, `?`→`$n` rebinding   |
| `dbdriver`          | DB driver name constants (`mysql`, `postgres`)                     |
| `http`              | chi router + middleware (auth, errors, standardized responses)     |
| `token`             | `token.Manager` implementation (`jwt.go`, HS256 JWT)               |
| `logger`            | structured logging (slog) setup                                    |
| `mailer`            | `mailer.MailSender` implementations: `log`, `smtp`, `ses` + MIME   |
| `queue`             | asynq client/server glue behind `queue.Enqueuer`                   |
| `sms`               | `sms.Sender` implementations: `log`, `twilio`                      |
| `storage`           | `storage.Storage` implementations: `local`, `s3` + factory         |

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

### Why `dbdriver` is separate from `database`

`dbdriver` holds only the driver-name constants (`mysql`, `postgres`). It
exists because `config` and `database` need the same constants, but each
already imports the other's types (`database` uses `config.DatabaseConfig`,
`config` validates the driver). Merging the constants into `database` would
create an import cycle (`config` ⇄ `database`). Keeping them in a tiny,
dependency-free package lets `config`, `database`, and module repositories all
reference one source of truth.
