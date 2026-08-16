# `internal/platform/clock/`

**Technical infrastructure** — a swappable time source for testability.
Production code uses `Real`; tests inject `Fixed` to pin the current time.

## Key types

| Symbol | Purpose                                                    |
|--------|------------------------------------------------------------|
| `Clock`| interface with `Now() time.Time`                           |
| `Real` | the production clock; `Loc` is the app timezone, nil means UTC |
| `Fixed`| a clock pinned to a specific instant (tests), normalized to UTC |

## Usage

```go
// production:
clk := clock.Real{Loc: cfg.Location()}

// tests:
clk := clock.Fixed{T: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
```

A zero `Real` is a valid, always-current clock (UTC).

## Dependency rules

Platform helper; imports stdlib only.
