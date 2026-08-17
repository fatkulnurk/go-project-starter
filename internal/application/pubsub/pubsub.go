// Package pubsub defines the cross-cutting publish/subscribe contract. Business
// modules publish messages and subscribe to topics; the concrete broker
// (memory, redis, rabbitmq, kafka) is hidden behind Publisher and Registrar.
package pubsub

import (
	"context"
)

// Message is a single event published to a topic. Payload is the opaque event
// body; the publisher never interprets it.
type Message struct {
	Topic   string
	Payload []byte
}

// Publisher broadcasts messages to a topic. All subscribers of that topic
// receive every message (fan-out); unlike the queue, there is no claiming or
// retry — delivery is fire-and-forget.
type Publisher interface {
	// Publish broadcasts a message to all subscribers of Message.Topic. It
	// returns an error when the broker rejects the message (e.g. broker
	// unavailable); it does not wait for subscribers to process it.
	Publish(ctx context.Context, m Message) error
}

// Handler processes a single message of a subscribed topic. Returning an error
// is logged by the subscriber and the message is dropped: pub/sub has no retry
// semantics by design (handlers should be idempotent).
type Handler func(ctx context.Context, topic string, payload []byte) error

// Registrar subscribes handlers to topics on a subscriber.
// A subscriber dispatches every message of a topic to its registered handler.
type Registrar interface {
	// Subscribe binds a topic to a handler. Registering the same topic twice
	// replaces the previous handler.
	Subscribe(topic string, h Handler)
}
