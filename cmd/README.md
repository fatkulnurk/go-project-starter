# cmd — Composition roots

Each binary here is an **independent composition root**: it reads configuration,
wires the platform adapters (DB, cache, queue, mail, SMS, storage, tokens) into
the business modules, and then runs its own entrypoint. No business logic lives
here — only dependency wiring.

## Binaries

| Command     | Purpose                                                                  |
| ----------- | ------------------------------------------------------------------------ |
| `api`       | HTTP API server (port `APP_PORT`, default 32100). Mounts auth and RBAC    |
|             | routes plus `/assets/*` static files.                                 |
| `web`       | Public web server (port `WEB_PORT`, default 32101). Serves the homepage   |
|             | landing page and `/assets/*` static files.                               |
| `worker`    | Queue worker. Processes async tasks enqueued by the API (email/SMS       |
|             | delivery, etc.).                                                         |
| `scheduler` | Periodic-job runner. Executes the scheduled jobs registered by modules   |
|             | (e.g. the homepage "tick" demo, logging the time every minute).          |
| `subscriber`| Pub/sub consumer. Runs the topic handlers registered by modules (e.g.    |
|             | the homepage "app.demo.ping" demo) against the configured broker.         |
| `migrate`   | Migration CLI. Applies/reverts the SQL migrations in `migrations/`.      |
| `seed`      | One-off seeder (like `artisan db:seed`). Creates the default roles/      |
|             | permissions and demo users defined in the module seeder packages         |
|             | (e.g. `auth/seeder.DefaultUsers`, `rbac/seeder.DefaultRoles`).           |

## Running

```sh
# API
go run ./cmd/api

# Web front
go run ./cmd/web

# Worker
go run ./cmd/worker

# Scheduler (periodic jobs)
go run ./cmd/scheduler

# Subscriber (pub/sub consumer) — needs a broker (PUBSUB_DRIVER)
go run ./cmd/subscriber

# Migrations
go run ./cmd/migrate up

# Seeder (idempotent, safe to re-run) — run after migrations, before the API
go run ./cmd/seed

# Seed only one seeder (like artisan db:seed --class)
go run ./cmd/seed auth.users
```

All binaries read the same environment (see `.env.example`). Start with the
migrations, then run whichever servers you need.

## Rules

- Wire dependencies here only; never instantiate platform adapters inside
  modules.
- Keep each binary lean — most real wiring lives in the `platform` packages.
- Tests for wiring logic belong beside the component, not here.
