# `internal/platform/queue/`

**Technical infrastructure** — the queue backends and their factories. Business
modules must only use the `internal/application/queue` contracts.

## Key types

| Symbol   | Purpose                                                    |
|----------|------------------------------------------------------------|
| `Client` | union of `queue.Enqueuer` + `Close()`                       |
| `Worker` | union of `queue.Registrar` + `Run()`/`Stop()` lifecycle     |
| `AsynqClient` / `AsynqServer` | hibiken/asynq backed (Redis)         |
| `DatabaseClient` / `DatabaseServer` | `queue_jobs` table backed          |

## Factories

- `NewClient(cfg config.QueueConfig, db *sql.DB, dbDriver string) (Client, error)` —
  selects the enqueuing client by `QUEUE_DRIVER` (`asynq` or `db`).
- `NewServer(cfg, log *slog.Logger, db, dbDriver) (Worker, error)` — selects the
  task-processing worker for the same drivers.

## Backends

- **asynq** — Redis; the client reuses `REDIS_ADDR`/`REDIS_PASSWORD` but runs in
  its own logical DB (`QUEUE_REDIS_DB`, default `1`) so cache and queue keys
  never collide. Concurrency from `QUEUE_CONCURRENCY`. `queue.ErrPermanent` maps
  to `asynq.SkipRetry`. The error handler never logs the payload (OTP / magic-link
  secrets must not leak).
- **db** — polls `queue_jobs` with a worker pool (`QUEUE_CONCURRENCY`, default
  10); jobs are claimed atomically via `SELECT ... FOR UPDATE`, retried with a
  growing backoff (5s → 5m), time-boxed to 5 minutes, and leases (default 6m)
  keep crashed workers' jobs reclaimable. Reuses the shared SQL pool.

## Usage

```go
client, err := queue.NewClient(cfg.Queue, db, cfg.Database.Driver)
defer client.Close()

worker, err := queue.NewServer(cfg.Queue, log, db, cfg.Database.Driver)
worker.Register("mail.send", handleMailSend)
go worker.Run()
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
