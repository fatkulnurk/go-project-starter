# `internal/platform/audit/`

**Technical infrastructure** — a SQL-backed implementation of the
`internal/application/audit` contract. It writes to the `audit_logs` table via
`database/sql`.

## Key types

| Symbol       | Purpose                                                    |
|--------------|------------------------------------------------------------|
| `SQLRecorder`| `audit.Recorder` implementation backed by a shared `*sql.DB` |

## Constructors

- `New(db *sql.DB, driver string, loc *time.Location) *SQLRecorder` — builds a
  recorder for the given pool. `driver` selects the placeholder dialect used by
  `database.Rebind`; `loc` sets the timezone for timestamps (UTC when nil).

## Behavior

- `Record(ctx, entry)` inserts one row; `OldValues`/`NewValues` are stored as
  JSON and an entry whose actor has no type is recorded as the system actor.
- Uses `id.New()` (UUID v7) for row ids and honors `APP_TIMEZONE` for
  timestamps.

## Usage

```go
recorder := audit.New(db, cfg.Database.Driver, cfg.Location())
// wire into commands:
audit.RecordBestEffort(ctx, recorder, audit.Entry{
    SubjectType: "user",
    SubjectID:   userID,
    Action:      audit.ActionUpdated,
    Actor:       audit.ActorFrom(ctx),
})
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
