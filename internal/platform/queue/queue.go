// Package queue provides the asynq-based queue backend. Business modules must
// only use the internal/application/queue contracts.
package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/hibiken/asynq"
)

// Client enqueues tasks to Redis via asynq.
type Client struct {
	c *asynq.Client
}

// NewClient builds an asynq client from config.
func NewClient(cfg config.QueueConfig) (*Client, error) {
	redisOpt, err := parseRedisOpt(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{c: asynq.NewClient(redisOpt)}, nil
}

// Enqueue implements queue.Enqueuer.
func (c *Client) Enqueue(ctx context.Context, t queue.Task) error {
	opts := make([]asynq.Option, 0, 1)
	if t.MaxRetry > 0 {
		opts = append(opts, asynq.MaxRetry(t.MaxRetry))
	}
	task := asynq.NewTask(t.Type, t.Payload, opts...)
	_, err := c.c.EnqueueContext(ctx, task)
	return err
}

// Close releases the client connection.
func (c *Client) Close() error { return c.c.Close() }

// Server runs task handlers registered by modules.
type Server struct {
	srv *asynq.Server
	mux *asynq.ServeMux
}

// NewServer builds an asynq server from config. The server logs through the
// provided slog logger, records failures via the error handler, and reports
// liveness through the health check.
func NewServer(cfg config.QueueConfig, log *slog.Logger) (*Server, error) {
	redisOpt, err := parseRedisOpt(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 10
	}
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: cfg.Concurrency,
		Queues:      map[string]int{"default": 3},
		Logger:      slogAdapter{log: log},
		LogLevel:    logLevel(log.Enabled(context.Background(), slog.LevelDebug)),
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			log.Error("task failed",
				"task_type", task.Type,
				"payload", string(task.Payload()),
				"err", err,
			)
		}),
		HealthCheckFunc: func(err error) {
			if err != nil {
				log.Error("queue health check failed", "err", err)
				return
			}
			log.Debug("queue health check ok")
		},
		HealthCheckInterval: 30 * time.Second,
		ShutdownTimeout:     10 * time.Second,
	})
	return &Server{srv: srv, mux: asynq.NewServeMux()}, nil
}

// Register implements queue.Registrar.
func (s *Server) Register(taskType string, h queue.TaskHandler) {
	s.mux.HandleFunc(taskType, func(ctx context.Context, t *asynq.Task) error {
		err := h(ctx, t.Payload())
		if errors.Is(err, queue.ErrPermanent) {
			return asynq.SkipRetry
		}
		return err
	})
}

// Run blocks and processes tasks until Stop is called.
func (s *Server) Run() error {
	if err := s.srv.Run(s.mux); err != nil {
		return fmt.Errorf("asynq server: %w", err)
	}
	return nil
}

// Stop gracefully stops the server.
func (s *Server) Stop() { s.srv.Shutdown() }

// slogAdapter adapts slog to asynq.Logger.
type slogAdapter struct {
	log *slog.Logger
}

func (a slogAdapter) Debug(args ...any) { a.log.Debug(formatArgs(args...)) }

func (a slogAdapter) Info(args ...any) { a.log.Info(formatArgs(args...)) }

func (a slogAdapter) Warn(args ...any) { a.log.Warn(formatArgs(args...)) }

func (a slogAdapter) Error(args ...any) { a.log.Error(formatArgs(args...)) }

func (a slogAdapter) Fatal(args ...any) { a.log.Error(formatArgs(args...)) }

func formatArgs(args ...any) string {
	msg, ok := args[0].(string)
	if !ok {
		return fmt.Sprint(args...)
	}
	return msg
}

func logLevel(debug bool) asynq.LogLevel {
	if debug {
		return asynq.DebugLevel
	}
	return asynq.InfoLevel
}

func parseRedisOpt(cfg config.QueueConfig) (asynq.RedisClientOpt, error) {
	if cfg.RedisAddr == "" {
		return asynq.RedisClientOpt{}, errors.New("queue redis address is empty")
	}
	return asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		PoolSize: cfg.RedisPoolSize,
	}, nil
}
