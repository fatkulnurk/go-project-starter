package pubsub

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/fatkulnurk/go-project-starter/internal/application/pubsub"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/twmb/franz-go/pkg/kgo"
)

// KafkaPublisher produces records to a topic. franz-go enables idempotent
// producing by default, so duplicate produce retries are tolerated; messages
// are durable in the broker log once acknowledged.
type KafkaPublisher struct {
	cl  *kgo.Client
	log *slog.Logger
}

// newKafkaPublisher builds a kgo producer from config.
func newKafkaPublisher(cfg config.KafkaConfig, log *slog.Logger) (*KafkaPublisher, error) {
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ClientID(cfg.ClientID),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka client: %w", err)
	}
	return &KafkaPublisher{cl: cl, log: log}, nil
}

// Publish implements pubsub.Publisher; the message is produced synchronously.
func (p *KafkaPublisher) Publish(ctx context.Context, m pubsub.Message) error {
	return p.cl.ProduceSync(ctx, &kgo.Record{Topic: m.Topic, Value: m.Payload}).FirstErr()
}

// Close flushs pending records and releases the client.
func (p *KafkaPublisher) Close() error {
	p.cl.Close()
	return nil
}

// KafkaSubscriber receives every published message: each instance joins a
// unique consumer group (GroupPrefix-InstanceID), so every instance gets every
// record = a true broadcast. Reuse a shared InstanceID to make instances
// compete instead. Offsets are committed automatically (at-least-once), so a
// handler error may be redelivered after restart — handlers should be
// idempotent.
type KafkaSubscriber struct {
	cl  *kgo.Client
	log *slog.Logger

	mu     sync.RWMutex
	subs   map[string]pubsub.Handler
	topics []string
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// newKafkaSubscriber builds a kgo client in the instance's own consumer group.
func newKafkaSubscriber(cfg config.KafkaConfig, instance string, log *slog.Logger) (*KafkaSubscriber, error) {
	group := cfg.GroupPrefix + "-" + resolveInstanceID(instance)
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ClientID(cfg.ClientID),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka client: %w", err)
	}
	return &KafkaSubscriber{cl: cl, log: log, subs: make(map[string]pubsub.Handler)}, nil
}

// Subscribe implements pubsub.Registrar.
func (s *KafkaSubscriber) Subscribe(topic string, h pubsub.Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs[topic] = h
	s.topics = append(s.topics, topic)
}

// Run implements Subscriber. It subscribes the client to every registered topic
// and polls until Stop.
func (s *KafkaSubscriber) Run() error {
	s.mu.RLock()
	topics := append([]string(nil), s.topics...)
	subs := make(map[string]pubsub.Handler, len(s.subs))
	for t, h := range s.subs {
		subs[t] = h
	}
	s.mu.RUnlock()

	if len(topics) == 0 {
		s.log.Warn("kafka pubsub: no topics subscribed, subscriber idle")
	} else {
		s.cl.AddConsumeTopics(topics...)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			fetches := s.cl.PollFetches(ctx)
			if ctx.Err() != nil {
				return
			}
			if errs := fetches.Errors(); len(errs) > 0 {
				for _, e := range errs {
					s.log.Error("kafka fetch error", "topic", e.Topic, "err", e.Err)
				}
				continue
			}
			fetches.EachRecord(func(rec *kgo.Record) {
				h := subs[rec.Topic]
				if h == nil {
					return
				}
				if err := h(ctx, rec.Topic, rec.Value); err != nil {
					s.log.Error("pubsub handler failed", "topic", rec.Topic, "err", err)
				}
			})
		}
	}()
	s.wg.Wait()
	s.cl.Close()
	return nil
}

// Stop implements Subscriber.
func (s *KafkaSubscriber) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}
