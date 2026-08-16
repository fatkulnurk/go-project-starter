# `internal/platform/id/`

**Technical infrastructure** — implements the `internal/application/id` contract
with time-ordered UUID v7 identifiers. The concrete generator is selected here,
never in application code.

## Key types

| Symbol      | Purpose                                                    |
|-------------|------------------------------------------------------------|
| `Generator` | stateless value receiver emitting version-7 UUID strings; a zero value is ready to use |

## Behavior

- `New()` returns a time-ordered UUID v7 string (sorts by insertion time, keeps
  indexes friendly). It panics if the underlying UUID generator is unavailable,
  which is not expected in practice.

## Usage

```go
// composition root, once:
id.SetDefault(id.Generator{})

// anywhere in the app:
userID := id.New()
```

## Dependency rules

Wraps `github.com/google/uuid`; never `internal/modules`.
