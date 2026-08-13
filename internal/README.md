# `internal/`

This tree holds **all application code**. It is private to the module —
nothing outside this repository may import it.

## Layout

```text
internal/
├── application/   cross-cutting capabilities (contracts)
├── modules/       business modules (auth, rbac, media, ...)
└── platform/      technical infrastructure (http, db, redis, ...)
```

## Dependency rules

Dependencies point **inward and downward**. They are enforced by
`arch.yaml` (go-arch-lint) and the grep check in `make check`.

```text
cmd/ ─────────────► internal/ ├── modules ──► internal/application
                                        │            │
internal/modules ────────────► internal/platform ────┘
```

- `internal/modules` may import `internal/application` and `internal/platform`
  (adapters), and other modules **only through their package root API**.
- `internal/application` may import nothing else from this tree.
- `internal/platform` may import anything in this tree, but never `modules`.
- `internal/modules/**/domain` stays pure: no SQL, no HTTP, no framework.

## Adding a feature

1. Does it span multiple business areas? Add a **contract** in
   `internal/application` and a driver in `internal/platform`.
2. Is it a business capability? Add a **module** in `internal/modules`.
3. Wire it in `cmd/` — never inside a module.

See `internal/modules/README.md` for the per-module structure.
