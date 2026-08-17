# `internal/platform/schedule/`

**Technical infrastructure** — the scheduled-jobs backend. Business modules
must only use the `internal/application/schedule` contracts.

## Key types

| Symbol     | Purpose                                                       |
|------------|---------------------------------------------------------------|
| `Worker`   | union of `schedule.Registrar` + `Run()`/`Stop()` lifecycle     |
| `tickerWorker` | stdlib `time.Ticker` backed implementation (unexported)   |

## Factory

- `New(log *slog.Logger) Worker` — builds the ticker-backed scheduler. `log`
  receives handler failures and invalid registrations.

## Behavior

- One goroutine per registered job; each handler runs at its `Job.Interval`.
- A non-positive `Interval` disables the job (logged, skipped).
- `Run()` blocks until `Stop()`; `Stop()` cancels the contexts of in-flight
  handlers and `Run()` returns only after they finish (like the db queue
  server).
- A handler error is logged and never stops the scheduler or sibling jobs.

## Usage

```go
sched := schedule.New(log)
homepageModule.RegisterSchedule(sched) // registers "homepage.tick" @ 1m
go sched.Run()
// ... on shutdown:
sched.Stop()
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
