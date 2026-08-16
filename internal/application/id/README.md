# `internal/application/id/`

**Cross-cutting capability** — the shared identifier generator contract. Modules
and platform packages call `id.New` here instead of a UUID library directly, so
the implementation choice stays centralized and the application layer stays free
of external libraries.

## Key types

| Symbol      | Purpose                                                    |
|-------------|------------------------------------------------------------|
| `Generator` | produces unique identifiers (`New() string`)               |

## Functions

- `New() string` — returns a new identifier via the configured generator.
  Panics when no generator has been set, surfacing a wiring mistake instead of
  silently minting non-UUID ids.
- `SetDefault(g Generator)` — installs the generator used by `New`. Call it once
  during startup, before any identifiers are minted.

## Usage

```go
// composition root, once at startup:
id.SetDefault(id.Generator{})

// anywhere:
modelID := id.New()
```

UUID v7 identifiers are time-ordered so rows sort naturally by insertion time,
which keeps indexes friendly.

## Dependency rules

Vendor-free contract; imports stdlib only. The implementation
(`internal/platform/id`) is selected once via `SetDefault`.
