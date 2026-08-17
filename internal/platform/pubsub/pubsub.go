// Package pubsub provides the pub/sub backends and their factories. Business
// modules must only use the internal/application/pubsub contracts.
package pubsub

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/fatkulnurk/go-project-starter/internal/application/pubsub"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// Publisher is the union of the application publisher contract and cleanup.
// Modules receive this from the composition root to broadcast events without
// knowing which broker is configured.
type Publisher interface {
	pubsub.Publisher

	// Close releases the broker connection.
	Close() error
}

// Subscriber is the union of topic registration and the run/stop lifecycle.
// The composition root hands it to modules so they can subscribe handlers
// before Run starts dispatching.
type Subscriber interface {
	pubsub.Registrar

	// Run blocks and dispatches messages until Stop is called.
	Run() error

	// Stop gracefully stops the subscriber.
	Stop()
}

// NewClient builds a publishing client for the configured broker. log receives
// diagnostics (e.g. unacknowledged publishes on the RabbitMQ backend).
func NewClient(cfg config.PubSubConfig, log *slog.Logger) (Publisher, error) {
	switch cfg.Driver {
	case config.DriverMemory:
		return memoryClient{}, nil
	case config.DriverRedis:
		return newRedisClient(cfg.Redis), nil
	case config.DriverRabbitMQ:
		return newRabbitMQPublisher(cfg.RabbitMQ, log)
	case config.DriverKafka:
		return newKafkaPublisher(cfg.Kafka, log)
	default:
		return nil, fmt.Errorf("unknown pubsub driver %q", cfg.Driver)
	}
}

// NewServer builds a subscriber for the configured broker. log receives
// handler failures and dispatch diagnostics.
func NewServer(cfg config.PubSubConfig, log *slog.Logger) (Subscriber, error) {
	switch cfg.Driver {
	case config.DriverMemory:
		return newMemorySubscriber(), nil
	case config.DriverRedis:
		return newRedisSubscriber(cfg.Redis, log), nil
	case config.DriverRabbitMQ:
		return newRabbitMQSubscriber(cfg.RabbitMQ, cfg.InstanceID, log)
	case config.DriverKafka:
		return newKafkaSubscriber(cfg.Kafka, cfg.InstanceID, log)
	default:
		return nil, fmt.Errorf("unknown pubsub driver %q", cfg.Driver)
	}
}

// resolveInstanceID returns a stable per-process identifier, falling back to a
// random 4-byte hex suffix when the config leaves it empty. RabbitMQ queue
// names and Kafka consumer-group IDs are built from it: a unique ID per
// subscriber instance yields a broadcast (every instance receives every
// message), a shared ID yields competing consumers.
func resolveInstanceID(cfg string) string {
	if cfg != "" {
		return cfg
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
