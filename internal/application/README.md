# `internal/application/`

**Cross-cutting capabilities** — the contracts that more than one module needs.
Each package here answers a question like "how do I send an email?", "who is
calling?", or "is this allowed?", without caring *how* it is implemented.

This layer is where the business modules and the technical platform meet. It
must not import `internal/platform` or `internal/modules`.

## What lives here

| Package         | Purpose                                                        |
|-----------------|----------------------------------------------------------------|
| `auth`          | identity contract: `Authenticator`, `Identity`, context helpers |
| `authorization` | `Authorizer` ("may this caller do this action?")               |
| `audit`         | polymorphic audit contract (`Auditor`, `Entry`, actor context) |
| `branding`      | shared brand data (`Common`) + template render helper          |
| `cache`         | cache contract (`Get`, `Set`, `Delete`, `Increment`, ...)      |
| `hash`          | password hashing contract                                       |
| `mailer`        | email contract (text, HTML, attachments)                       |
| `media`         | media-library contract (`media.Library`, attach/read/remove/URL)|
| `otp`           | one-time-passcode generation/validation helpers                |
| `queue`         | enqueue + task registration contract (asynq)                   |
| `sms`           | SMS contract                                                    |
| `storage`       | object-storage contract (write, read, delete, presign)         |
| `token`         | access-token contract (issue, parse)                            |
| `apierr`        | shared API error sentinels + HTTP status/code mapping           |

## What belongs here

- **Interfaces** describing *what* a capability does, with no implementation
  detail and no vendor imports (no chi, no redis, no aws, no asynq).
- **Domain-neutral helpers** (e.g. OTP generation) shared by several modules.
- **Sentinel errors** every module agrees on, so the HTTP layer can map them
  to one consistent response shape.

## What does NOT belong here

- SQL, structs mapped to database rows, or drivers.
- HTTP handlers, routers, middleware.
- Business rules of a single module (e.g. "a user must verify before login") —
  those go in the module's `domain`/`application`.

Rule of thumb: if only one module needs it, keep it inside that module. If two
or more modules need it, move the *contract* here and the *implementation* to
`internal/platform`.
