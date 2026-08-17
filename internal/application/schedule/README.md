# `internal/application/schedule/`

**Cross-cutting capability** — the scheduled-jobs contract. Business modules
register periodic jobs and the concrete backend (cron parser, ticker, ...) is
hidden behind `Registrar`.

## Key types

| Symbol       | Purpose                                              |
|--------------|------------------------------------------------------|
| `Job`        | a unit of periodic work: `Name`, `Cron`, `Handler`   |
| `JobHandler` | runs a single execution of a job (`func(ctx) error`) |
| `Registrar`  | registers periodic jobs on a scheduler               |

`JobHandler` receives a `context.Context` that is cancelled when the scheduler
stops, so long-running handlers can abort early. An error returned by a handler
is logged by the scheduler and does not stop the job.

`Cron` is a 5-field cron expression: `minute hour day-of-month month
day-of-week` (e.g. `0 3 * * FRI` for every Friday at 03:00). It is evaluated in
the scheduler's configured timezone.

## Usage

```go
// register (scheduler side):
registrar.Register(schedule.Job{
    Name:    "homepage.tick",
    Cron:    "* * * * *",
    Handler: func(ctx context.Context) error {
        slog.Info("tick", "time", time.Now())
        return nil
    },
})
```

Implemented by `internal/platform/schedule` (`New`).

## Dependency rules

Vendor-free contract; imports stdlib only.
