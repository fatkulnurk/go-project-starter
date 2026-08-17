package pubsub

import (
	"context"
	"log/slog"
	"sync"

	"github.com/fatkulnurk/go-project-starter/internal/application/pubsub"
)

// memBus is the package-level in-process bus shared by memory publishers and
// subscribers. It only works within a single process: a publisher and a
// subscriber in different binaries cannot see each other. Use it for tests and
// single-process development, never for cross-process fan-out.
var memBus = &memoryBus{subs: make(map[string]pubsub.Handler)}

// memoryBus fans published messages out to the currently subscribed handlers.
type memoryBus struct {
	mu   sync.RWMutex
	subs map[string]pubsub.Handler
}

func (b *memoryBus) set(topic string, h pubsub.Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[topic] = h
}

// dispatch delivers a message to its handler asynchronously. Messages with no
// subscriber are dropped, mirroring the fire-and-forget contract.
func (b *memoryBus) dispatch(ctx context.Context, m pubsub.Message) {
	b.mu.RLock()
	h, ok := b.subs[m.Topic]
	b.mu.RUnlock()
	if !ok {
		return
	}
	go func() {
		if err := h(context.WithoutCancel(ctx), m.Topic, m.Payload); err != nil {
			slog.Error("pubsub handler failed", "topic", m.Topic, "err", err)
		}
	}()
}

// memoryClient publishes to the shared in-process bus.
type memoryClient struct{}

func (memoryClient) Publish(_ context.Context, m pubsub.Message) error {
	memBus.dispatch(context.Background(), m)
	return nil
}

func (memoryClient) Close() error { return nil }

// memorySubscriber subscribes handlers to the shared in-process bus and
// blocks until Stop.
type memorySubscriber struct {
	once sync.Once
	stop chan struct{}
}

// newMemorySubscriber builds a memory subscriber.
func newMemorySubscriber() *memorySubscriber {
	return &memorySubscriber{stop: make(chan struct{})}
}

func (m *memorySubscriber) Subscribe(topic string, h pubsub.Handler) {
	memBus.set(topic, h)
}

func (m *memorySubscriber) Run() error {
	<-m.stop
	return nil
}

func (m *memorySubscriber) Stop() {
	m.once.Do(func() { close(m.stop) })
}
