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
├── templates/         embedded message/page templates, split by channel
│   ├── web/           HTML pages rendered by web/api handlers
│   ├── email/         email HTML + text (subject/body)
│   └── sms/           SMS body text
├── adapters/
│   ├── api/           HTTP handlers + DTOs for JSON APIs (no business logic)
│   ├── web/           HTTP handlers rendering HTML pages (non-JSON, e.g. homepage)
│   └── queue/         background task handlers (if the module has any)
├── module.go          `New(Dependencies) *Module` — composition inside module
├── api.go             `API` + `Service` — the public face other modules may use
└── doc.go             package documentation
```

`module.go` is the composition root *inside* the module: it takes dependencies
(ports) and builds the `API` (use cases) plus the module's exported helpers
(`Authenticator()`, `RegisterAPI()`, `RegisterWeb()`, `RegisterQueue()`).

### Content templates vs. platform layouts

Each module keeps its own content templates in a root `templates/` package,
split by channel (`web/` for pages, `email/` for email HTML + text, `sms/` for
SMS bodies). Those are the *what* — module-owned copy. The *how* — shared
layouts — lives in `internal/platform` (`view` for pages, `mailer` for email)
and is reused via `ParseFS`.

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
- **adapters/api**: only parses requests, calls one use case, renders the
  standardized response. Never contains business rules or SQL. Use `web/`
  instead when a module serves rendered HTML pages rather than a JSON API.

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
3. Implement repositories in `infrastructure/`, handlers in `adapters/api/`.
4. Expose use cases through `api.go` and anything other modules need as a
   narrow `Service` interface.
5. Wire it in `cmd/api/main.go` (and `cmd/worker/main.go` if it has tasks).
6. Add migrations under `migrations/` and update `README.md`.
