# `internal/platform/logger/`

**Technical infrastructure** — wraps `log/slog` with context-aware request
logging.

## Key functions

- `New(environment string) *slog.Logger` — builds the application-wide logger.
  Production emits JSON to stdout for easy ingestion; development uses
  human-readable text at debug level.
- `With(ctx, logger) context.Context` — stores the logger inside the context.
- `From(ctx) *slog.Logger` — returns the logger stored in `ctx`, or
  `slog.Default()` when the context carries none (or a nil one).

## Usage

```go
log := logger.New(cfg.Environment)
ctx := logger.With(r.Context(), log)

logger.From(ctx).Info("job done", "task_type", "mail.send")
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
