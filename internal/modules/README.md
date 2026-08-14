# `internal/modules/`

**Business modules** — the *what* of the system. Each folder is one
independently-testable business capability (auth, rbac, media, ...). A module
is a vertical slice: its own domain, use cases, persistence, and HTTP/queue
adapters, all wired together in `cmd/`.

## The structure of a module

```text
internal/modules/<name>/
├── domain/            pure entities, value objects, repo interfaces, sentinels
├── application/       use cases: commands/, queries/ (+ module-local ports)
├── infrastructure/    repository implementations (database/sql)
├── adapters/
│   ├── httpapi/       HTTP handlers + DTOs (no business logic)
│   │   └── templates/ embedded HTML pages rendered by these handlers
│   └── queue/         background task handlers (if the module has any)
│       └── templates/ embedded email/SMS message templates
├── module.go          `New(Dependencies) *Module` — composition inside module
├── api.go             `API` + `Service` — the public face other modules may use
└── doc.go             package documentation
```

`module.go` is the composition root *inside* the module: it takes dependencies
(ports) and builds the `API` (use cases) plus the module's exported helpers
(`Authenticator()`, `RegisterHTTP()`, `RegisterQueue()`).

### Content templates vs. platform layouts

Each adapter that renders content keeps its own templates in an
`adapters/<adapter>/templates/` subpackage: HTML pages under `httpapi/templates/`,
email/SMS messages under `queue/templates/`. Those are the *what* — module-owned
copy. The *how* — shared layouts — lives in `internal/platform` (`view` for
pages, `mailer` for email) and is reused via `ParseFS`.

### Uniform skeleton

Every module keeps the same folder skeleton (`domain/`, `application/`,
`infrastructure/`, `adapters/`, `module.go`, `api.go`, `doc.go`) even when a
layer has no code yet, so navigation is predictable. A layer that genuinely has
no content documents that with a short package comment in a `doc.go` (see
`homepage/`).

## Dependency rules inside a module

```text
adapters/ ──► application/ ──► domain/
infrastructure/ ──► domain/
```

- **domain**: pure Go, no SQL/HTTP/frameworks, no imports from this tree.
- **application**: imports `domain`, `internal/application` (contracts), and
  `internal/platform/clock`. No drivers, no `adapters`.
- **infrastructure**: implements the domain repository interfaces with
  `database/sql` (+ `internal/platform/database`). SQL uses `?` placeholders,
  portability handled by `database.Rebind`.
- **adapters/httpapi**: only parses requests, calls one use case, renders the
  standardized response. Never contains business rules or SQL.

## Rules between modules

- A module may depend on another module **only through its package root API**
  (`internal/modules/<other>`), never its internals.
- Cross-cutting needs (email, cache, storage, ...) must be expressed as
  contracts in `internal/application` and satisfied in `cmd/`, not by importing
  another module's adapters.

## Adding a new module

1. Create the folder skeleton above.
2. Write `domain/` first (entities + repository interfaces), then
   `application/` use cases against those interfaces.
3. Implement repositories in `infrastructure/`, handlers in `adapters/httpapi/`.
4. Expose use cases through `api.go` and anything other modules need as a
   narrow `Service` interface.
5. Wire it in `cmd/api/main.go` (and `cmd/worker/main.go` if it has tasks).
6. Add migrations under `migrations/` and update `README.md`.
