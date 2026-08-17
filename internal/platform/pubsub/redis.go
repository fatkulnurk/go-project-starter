package pubsub

import (
	"context"
	"log/slog"
	"sync"

	"github.com/fatkulnurk/go-project-starter/internal/application/pubsub"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/redis/go-redis/v9"
)

// RedisClient broadcasts messages via Redis pub/sub. Redis is fire-and-forget:
// a message published when no subscriber is connected is lost. Suitable for
// low-criticality notifications, not for events that must be delivered.
type RedisClient struct {
	client *redis.Client
}

// newRedisClient builds a go-redis client from the shared Redis settings.
func newRedisClient(cfg config.RedisConfig) *RedisClient {
	return &RedisClient{client: redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})}
}

// Publish implements pubsub.Publisher.
func (c *RedisClient) Publish(ctx context.Context, m pubsub.Message) error {
	return c.client.Publish(ctx, m.Topic, m.Payload).Err()
}

// Close releases the client connection.
func (c *RedisClient) Close() error { return c.client.Close() }

// RedisSubscriber subscribes to topics on a single Redis connection and
// dispatches messages to the registered handlers. go-redis reconnects
// automatically, but messages published while disconnected are lost.
type RedisSubscriber struct {
	client *redis.Client
	log    *slog.Logger

	mu     sync.RWMutex
	subs   map[string]pubsub.Handler
	cancel context.CancelFunc
}

// newRedisSubscriber builds the subscriber.
func newRedisSubscriber(cfg config.RedisConfig, log *slog.Logger) *RedisSubscriber {
	return &RedisSubscriber{
		client: newRedisClient(cfg).client,
		log:    log,
		subs:   make(map[string]pubsub.Handler),
	}
}

// Subscribe implements pubsub.Registrar.
func (s *RedisSubscriber) Subscribe(topic string, h pubsub.Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs[topic] = h
}

// Run implements Subscriber. It subscribes to all registered topics and blocks
// until Stop.
func (s *RedisSubscriber) Run() error {
	s.mu.RLock()
	topics := make([]string, 0, len(s.subs))
	for t := range s.subs {
		topics = append(topics, t)
	}
	s.mu.RUnlock()

	if len(topics) == 0 {
		s.log.Warn("redis pubsub: no topics subscribed, subscriber idle")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	ps := s.client.Subscribe(ctx, topics...)
	defer ps.Close()

	ch := ps.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			s.mu.RLock()
			h := s.subs[msg.Channel]
			s.mu.RUnlock()
			if h == nil {
				continue
			}
			if err := h(ctx, msg.Channel, []byte(msg.Payload)); err != nil {
				s.log.Error("pubsub handler failed", "topic", msg.Channel, "err", err)
			}
		}
	}
}

// Stop implements Subscriber.
func (s *RedisSubscriber) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}
