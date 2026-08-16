# `internal/application/seed/`

**Cross-cutting capability** — the seeding contract, mirroring Laravel's
`database/seeders`: each seeder is a named unit of idempotent data
initialization exposing `Run(ctx)`, and a registry (like `DatabaseSeeder`) runs
them all or a named subset.

## Key types

| Symbol    | Purpose                                                    |
|-----------|------------------------------------------------------------|
| `Seed`    | a named unit of data initialization; `Run` must be idempotent and safe to re-run |
| `Registry`| collects seeders and runs them in registration order        |

## Functions

- `New() *Registry` — builds an empty registry.
- `Register(name, s)` — stores a seeder under a stable name; duplicate names
  panic (a wiring bug surfaced at startup).
- `Run(ctx)` — executes every registered seeder in registration order
  (`artisan db:seed`).
- `RunOnly(ctx, names...)` — executes the named seeders
  (`artisan db:seed --class=X`); unknown names fail loudly.

## Usage

```go
registry := seed.New()
registry.Register("rbac", rbacSeeder)
registry.Register("auth", authSeeder)

// composition root / cmd/seed:
if err := registry.Run(ctx); err != nil { /* abort */ }
```

Seeders must skip existing rows rather than fail, so re-running is safe.

## Dependency rules

Vendor-free contract; imports stdlib only.
