# `internal/application/pubsub/`

**Cross-cutting capability** — the publish/subscribe contract. Business modules
only publish messages and subscribe to topics; the concrete broker (`memory`,
`redis`, `rabbitmq`, `kafka`) is hidden behind `Publisher` and `Registrar`.

## Key types

| Symbol       | Purpose                                                    |
|--------------|------------------------------------------------------------|
| `Message`    | a broadcast event: `Topic`, `Payload` (`[]byte`)           |
| `Publisher`  | broadcasts messages to a topic (`Publish`)                 |
| `Registrar`  | binds a topic to its `Handler` on a subscriber (`Subscribe`)|
| `Handler`    | processes a single message of a topic                      |

## Semantics (differs from the queue)

| Queue (`internal/application/queue`) | Pub/Sub |
|--------------------------------------|---------|
| one task → **one** worker (competing consumers) | one message → **all** subscribers (fan-out) |
| at-least-once with retry (`MaxRetry`) | fire-and-forget, **no retry/ack** |
| `Enqueue` + `Register(taskType)`      | `Publish` + `Subscribe(topic)` |

A handler error is logged by the subscriber and the message is dropped. Because
pub/sub does not retry, handlers should be idempotent.

## Usage

```go
// publish (producer side):
err := publisher.Publish(ctx, pubsub.Message{
    Topic:   "app.demo.ping",
    Payload: []byte(`{"at":"..."}`),
})

// subscribe (subscriber side, runs in cmd/subscriber):
registrar.Subscribe("app.demo.ping", func(ctx context.Context, topic string, p []byte) error {
    slog.Info("received", "topic", topic, "payload", string(p))
    return nil
})
```

Implemented by `internal/platform/pubsub` (`NewClient`/`NewServer` with `memory`,
`redis`, `rabbitmq` or `kafka` backends).

## Dependency rules

Vendor-free contract; imports stdlib only.