# `internal/application/queue/`

**Cross-cutting capability** — the queue contract. Business modules only enqueue
tasks and register handlers; the concrete backend (`asynq`, `db`) is hidden
behind `Enqueuer` and `Registrar`.

## Key types

| Symbol       | Purpose                                                    |
|--------------|------------------------------------------------------------|
| `Task`       | a unit of async work: `Type`, `Payload` (`[]byte`), `MaxRetry` (0 = no retries) |
| `Enqueuer`   | pushes tasks onto the queue                                 |
| `Registrar`  | binds a task `Type` to its `TaskHandler` on a worker        |
| `TaskHandler`| processes a single task payload                             |

`ErrPermanent` signals a corrupt/unprocessable payload — do not retry. Any other
error is retried by the backend up to the task's `MaxRetry`.

## Usage

```go
// enqueue (API side):
err := enqueuer.Enqueue(ctx, queue.Task{
    Type:     "mail.send",
    Payload:  payloadBytes,
    MaxRetry: 3,
})

// register (worker side):
registrar.Register("mail.send", func(ctx context.Context, p []byte) error {
    return deliver(ctx, p)
})
```

Implemented by `internal/platform/queue` (`NewClient`/`NewServer` with `asynq`
or `db` backends).

## Dependency rules

Vendor-free contract; imports stdlib only.
