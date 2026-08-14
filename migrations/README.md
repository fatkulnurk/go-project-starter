# Migrations

SQL migrations for the project, applied with [`golang-migrate`](https://github.com/golang-migrate/migrate).
Each change is a pair of files:

- `NNNN_name.up.sql` — applied when migrating forward
- `NNNN_name.down.sql` — rolled back when migrating backward

## Usage

```sh
# bring the database up to the latest migration
go run ./cmd/migrate up

# roll back one migration
go run ./cmd/migrate down

# show the current version + dirty state
go run ./cmd/migrate version
```

The source is `file://migrations`; the target database is selected from the
`DB_*` environment variables (see `.env.example`).

## Adding a migration

1. Pick the next sequence number (e.g. `0010`).
2. Create `0010_description.up.sql` and `0010_description.down.sql`.
3. Follow existing conventions:
   - Plain SQL, one table per migration where possible.
   - `IF NOT EXISTS` / `IF EXISTS` where supported by both drivers.
   - Postgres and MySQL use the same SQL here (`?` placeholders are only a Go
     concern, not present in migration files).
4. Run `go run ./cmd/migrate up` and `down` locally to verify both directions.

## Conventions

- Prefix every migration with its zero-padded sequence number.
- Never edit an already-applied migration; add a new one instead.
- The `down` file must reverse exactly what `up` created.
