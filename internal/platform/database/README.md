# `internal/platform/database/`

**Technical infrastructure** — opens a `database/sql` connection for the
configured driver (`mysql` or `postgres`) and exposes helpers to keep queries
portable.

## Key functions

- `New(cfg config.DatabaseConfig) (*sql.DB, error)` — opens a pool for
  `cfg.Driver`, applies the configured pool sizes, and pings the server; a
  failed ping closes the pool and returns a wrapped error. Callers must `Close`
  the returned pool.
- `MigrateURL(cfg) string` — builds a golang-migrate database URL for `cfg`;
  credentials are escaped for the URL syntax.
- `Rebind(query, driver) string` — converts `?` placeholders to PostgreSQL `$1`
  style when `driver` is postgres; for mysql it returns the query unchanged. This
  keeps repository SQL written once with `?` placeholders.

## Behavior

- Applies `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`,
  `DB_CONN_MAX_IDLE_TIME`.
- Honors `APP_TIMEZONE`: MySQL sessions use the `time_zone`/`loc` params; the
  postgres URL sets `timezone`. `DB_SSL_MODE` maps to the driver-specific TLS
  settings (`disable`/`require`/`verify-ca`/`verify-full` for postgres,
  `disable`/`require`/`skip-verify` for mysql).

## Usage

```go
db, err := database.New(cfg.Database)
if err != nil { return err }
defer db.Close()

rows, err := db.QueryContext(ctx, database.Rebind("SELECT * FROM users WHERE id = ?", cfg.Database.Driver), id)
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`.
