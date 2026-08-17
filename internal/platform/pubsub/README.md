# `internal/platform/pubsub/`

**Technical infrastructure** — the pub/sub backends and their factories. Business
modules must only use the `internal/application/pubsub` contracts.

## Key types

| Symbol     | Purpose                                                    |
|------------|------------------------------------------------------------|
| `Publisher`| union of `pubsub.Publisher` + `Close()`                    |
| `Subscriber`| union of `pubsub.Registrar` + `Run()`/`Stop()` lifecycle  |
| `memoryClient` / `memorySubscriber` | in-process bus (single process)      |
| `RedisClient` / `RedisSubscriber`   | go-redis pub/sub                  |
| `RabbitMQPublisher` / `RabbitMQSubscriber` | amqp091-go (topic exchange)   |
| `KafkaPublisher` / `KafkaSubscriber` | twmb/franz-go (kgo)              |

## Factories

- `NewClient(cfg config.PubSubConfig, log *slog.Logger) (Publisher, error)` —
  selects the publishing client by `PUBSUB_DRIVER`.
- `NewServer(cfg config.PubSubConfig, log *slog.Logger) (Subscriber, error)` —
  selects the subscriber for the same drivers. No database is needed for any
  driver (unlike the queue's `db` backend).

## Backends & production posture

| Driver   | Durability            | Broadcast model                            | Production notes |
|----------|-----------------------|--------------------------------------------|------------------|
| `memory` | none                  | shared in-process bus                       | dev/test only, single process — a publisher and subscriber in different binaries cannot talk |
| `redis`  | none (fire-and-forget)| one connection fans out to all subscribers  | messages published with no subscriber are **lost**; reconnect gaps lose messages. Fine for low-criticality notifications |
| `rabbitmq`| optional (durable queues)| per-instance queue `pubsub.<instance>.<topic>` bound to a durable topic exchange | publisher confirms; auto-recovery; handler errors are dropped (not requeued). Best-effort durable broadcast |
| `kafka`  | durable log            | unique consumer group per instance (`<GroupPrefix>-<InstanceID>`) | every instance gets every record = true broadcast; auto-committed offsets (at-least-once → duplicates possible, keep handlers idempotent); replay possible from the log |

`PUBSUB_INSTANCE_ID` (default: random per process) disambiguates subscriber
instances. A unique ID per instance gives **broadcast**; a shared ID makes
instances **compete** for messages instead (RabbitMQ shares its queue, Kafka
shares its consumer group).

## Usage

```go
client, err := pubsub.NewClient(cfg.PubSub, log) // producer side (e.g. cmd/scheduler)
defer client.Close()
err = client.Publish(ctx, pubsub.Message{Topic: "app.demo.ping", Payload: p})

subscriber, err := pubsub.NewServer(cfg.PubSub, log) // consumer side (cmd/subscriber)
subscriber.Subscribe("app.demo.ping", onPing)
go subscriber.Run()
subscriber.Stop()
```

## Dependency rules

May import `internal/application` and other `internal/platform` packages, never
`internal/modules`. Broker libraries (go-redis, amqp091-go, franz-go) live only
here.