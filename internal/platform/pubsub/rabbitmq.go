package pubsub

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/pubsub"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

const confirmTimeout = 5 * time.Second

// RabbitMQPublisher publishes to a durable topic exchange with publisher
// confirms. Routing key == topic; durable (topic and messages) survive broker
// restarts. amqp091-go reconnects automatically on connection loss.
type RabbitMQPublisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
	log      *slog.Logger
	confirms chan amqp.Confirmation
}

// newRabbitMQPublisher dials the broker and declares the topic exchange.
func newRabbitMQPublisher(cfg config.RabbitMQConfig, log *slog.Logger) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	if err := ch.ExchangeDeclare(cfg.Exchange, "topic", true, false, false, false, nil); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq exchange declare: %w", err)
	}
	if err := ch.Confirm(false); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq publisher confirms: %w", err)
	}
	return &RabbitMQPublisher{
		conn:     conn,
		ch:       ch,
		exchange: cfg.Exchange,
		log:      log,
		confirms: ch.NotifyPublish(make(chan amqp.Confirmation, 1)),
	}, nil
}

// Publish implements pubsub.Publisher. It waits up to confirmTimeout for the
// broker acknowledgment; a timeout is logged but treated as delivered so a
// slow broker cannot stall callers.
func (p *RabbitMQPublisher) Publish(ctx context.Context, m pubsub.Message) error {
	err := p.ch.PublishWithContext(ctx, p.exchange, m.Topic, false, false, amqp.Publishing{
		ContentType:  "application/octet-stream",
		DeliveryMode: amqp.Persistent,
		Body:         m.Payload,
	})
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c := <-p.confirms:
		if !c.Ack {
			return errors.New("rabbitmq publish not acknowledged")
		}
		return nil
	case <-time.After(confirmTimeout):
		p.log.Warn("rabbitmq publish confirmation timed out", "topic", m.Topic)
		return nil
	}
}

// Close releases the channel and connection.
func (p *RabbitMQPublisher) Close() error {
	_ = p.ch.Close()
	return p.conn.Close()
}

// RabbitMQSubscriber receives every published message: each instance declares
// its own queue (pubsub.<instance>.<topic>) bound to the topic exchange, so
// fan-out is a true broadcast. With Durable queues, messages persist until the
// instance consumes them; otherwise queues are ephemeral (like Redis, messages
// published while an instance is down are lost). Handler errors are dropped,
// not requeued — pub/sub has no retry.
type RabbitMQSubscriber struct {
	cfg      config.RabbitMQConfig
	instance string
	log      *slog.Logger

	mu       sync.RWMutex
	subs     map[string]pubsub.Handler
	bindings []binding

	conn *amqp.Connection
	ch   *amqp.Channel

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// binding ties a topic to the queue that delivers it.
type binding struct {
	topic string
	queue string
}

// newRabbitMQSubscriber dials the broker and declares the topic exchange.
func newRabbitMQSubscriber(cfg config.RabbitMQConfig, instance string, log *slog.Logger) (*RabbitMQSubscriber, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	if err := ch.ExchangeDeclare(cfg.Exchange, "topic", true, false, false, false, nil); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq exchange declare: %w", err)
	}
	return &RabbitMQSubscriber{
		cfg:      cfg,
		instance: resolveInstanceID(instance),
		log:      log,
		subs:     make(map[string]pubsub.Handler),
		conn:     conn,
		ch:       ch,
	}, nil
}

// Subscribe implements pubsub.Registrar. It declares the per-instance queue for
// the topic and binds it to the exchange with the topic as routing key.
func (s *RabbitMQSubscriber) Subscribe(topic string, h pubsub.Handler) {
	name := fmt.Sprintf("pubsub.%s.%s", s.instance, topic)
	// Durable queues must be non-exclusive and persist; ephemeral queues are
	// exclusive and auto-deleted with the connection.
	q, err := s.ch.QueueDeclare(name, s.cfg.Durable, !s.cfg.Durable, !s.cfg.Durable, false, nil)
	if err != nil {
		s.log.Error("rabbitmq queue declare failed, subscription disabled", "topic", topic, "err", err)
		return
	}
	if err := s.ch.QueueBind(q.Name, topic, s.cfg.Exchange, false, nil); err != nil {
		s.log.Error("rabbitmq queue bind failed, subscription disabled", "topic", topic, "err", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs[topic] = h
	s.bindings = append(s.bindings, binding{topic: topic, queue: q.Name})
}

// Run implements Subscriber. It consumes one queue per subscribed topic and
// blocks until Stop.
func (s *RabbitMQSubscriber) Run() error {
	s.mu.RLock()
	binds := append([]binding(nil), s.bindings...)
	s.mu.RUnlock()

	if len(binds) == 0 {
		s.log.Warn("rabbitmq pubsub: no topics subscribed, subscriber idle")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	if err := s.ch.Qos(1, 0, false); err != nil {
		s.log.Warn("rabbitmq prefetch setup failed", "err", err)
	}
	for _, b := range binds {
		s.wg.Add(1)
		go s.consume(ctx, b)
	}
	s.wg.Wait()
	return nil
}

// consume reads deliveries from one queue and dispatches them to the topic's
// handler. Success acks; failure logs and drops (no requeue, no retry).
func (s *RabbitMQSubscriber) consume(ctx context.Context, b binding) {
	defer s.wg.Done()
	msgs, err := s.ch.Consume(b.queue, "", false, false, false, false, nil)
	if err != nil {
		s.log.Error("rabbitmq consume failed", "queue", b.queue, "err", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-msgs:
			if !ok {
				return
			}
			s.mu.RLock()
			h := s.subs[b.topic]
			s.mu.RUnlock()
			if h == nil {
				_ = d.Ack(false)
				continue
			}
			if err := h(ctx, b.topic, d.Body); err != nil {
				s.log.Error("pubsub handler failed", "topic", b.topic, "err", err)
				_ = d.Nack(false, false)
				continue
			}
			_ = d.Ack(false)
		}
	}
}

// Stop implements Subscriber.
func (s *RabbitMQSubscriber) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}
