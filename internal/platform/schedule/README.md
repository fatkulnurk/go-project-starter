# `internal/platform/schedule/`

**Technical infrastructure** — the scheduled-jobs backend. Business modules
must only use the `internal/application/schedule` contracts.

## Key types

| Symbol      | Purpose                                                     |
|-------------|-------------------------------------------------------------|
| `Worker`    | union of `schedule.Registrar` + `Run()`/`Stop()` lifecycle   |
| `Spec`      | compiled 5-field cron expression (`Parse`, `Match`)          |
| `cronWorker` | stdlib-backed implementation (unexported)                 |

## Factory

- `New(log *slog.Logger, loc *time.Location) Worker` — builds the cron-backed
  scheduler. `loc` is the timezone cron expressions are evaluated in (pass
  `cfg.Location()`); `log` receives handler failures and invalid registrations.

## Behavior

- One goroutine per registered job; each handler runs once per minute when its
  `Job.Cron` expression matches the current minute (evaluated in `loc`).
- A cron expression that fails to parse disables the job (logged, skipped).
- `Run()` blocks until `Stop()`; `Stop()` cancels the contexts of in-flight
  handlers and `Run()` returns only after they finish (like the db queue
  server).
- A handler error is logged and never stops the scheduler or sibling jobs.

## Cron expressions

Five fields, evaluated in the scheduler's location:

```text
* * * * *   every minute
0 3 * * *   every day at 03:00
0 3 * * FRI every Friday at 03:00
*/15 * * * * every 15 minutes
0 0 1 1 *   every Jan 1st at midnight
```

Each field supports `*`, `?` (alias for `*`), `*/n`, `a-b`, `a-b/n`, comma
lists, single values, and month/day-of-week names (`JAN..DEC`, `SUN..SAT`).
Day of week `0` and `7` are both Sunday. Implemented with stdlib only.

## Usage

```go
sched := schedule.New(log, cfg.Location())
homepageModule.RegisterSchedule(sched) // registers "homepage.tick" every minute
go sched.Run()
// ... on shutdown:
sched.Stop()
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
