# `internal/application/audit/`

**Cross-cutting capability** — the audit contract: recording *who did what to
which record*. Implementations live behind the `Recorder` interface (SQL, file,
cloud), so business modules never depend on a storage library.

## Key types

| Symbol      | Purpose                                                        |
|-------------|----------------------------------------------------------------|
| `Recorder`  | persists one `Entry` to the audit trail                         |
| `Entry`     | a single audited change: polymorphic subject + old/new values + `Actor` |
| `Actor`     | who performed the change (type, id, IP address, user agent)     |
| `Action`    | `created` / `updated` / `deleted` (stored verbatim)             |
| `ActorType` | `user` or `system` (workers, internal jobs)                     |

## Helpers

- `RecordBestEffort(ctx, r, entry)` — persists an entry, logging a warning on
  failure so a broken audit trail never breaks the business operation. Nil
  recorders are a no-op.
- `WithActor(ctx, actor)` / `ActorFrom(ctx)` — stash and recover the current
  actor from the context; middleware uses it so downstream commands can attach
  the caller to entries.

## Usage

```go
audit.RecordBestEffort(ctx, recorder, audit.Entry{
    SubjectType: "user",
    SubjectID:   userID,
    Action:      audit.ActionUpdated,
    OldValues:   map[string]any{"name": oldName},
    NewValues:   map[string]any{"name": newName},
    Actor:       audit.ActorFrom(ctx),
})
```

Implemented by `internal/platform/audit` (`SQLRecorder`), which the composition
root wires in.

## Dependency rules

Vendor-free contract; may import only stdlib and other `internal/application`
packages. No SQL, no HTTP.
