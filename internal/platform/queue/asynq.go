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

// AsynqClient enqueues tasks to Redis via asynq.
// It wraps an asynq.Client and is safe for concurrent use by multiple modules.
type AsynqClient struct {
	c *asynq.Client
}

// newAsynqClient builds an asynq client from config.
func newAsynqClient(cfg config.QueueConfig) (*AsynqClient, error) {
	redisOpt, err := parseRedisOpt(cfg)
	if err != nil {
		return nil, err
	}
	return &AsynqClient{c: asynq.NewClient(redisOpt)}, nil
}

// Enqueue implements queue.Enqueuer. MaxRetry 0 means no retries; the option
// is always set so the asynq default of 25 does not silently apply.
func (c *AsynqClient) Enqueue(ctx context.Context, t queue.Task) error {
	task := asynq.NewTask(t.Type, t.Payload, asynq.MaxRetry(t.MaxRetry))
	_, err := c.c.EnqueueContext(ctx, task)
	return err
}

// Close releases the client connection.
// It closes the underlying asynq client and returns the close error, if any.
func (c *AsynqClient) Close() error { return c.c.Close() }

// AsynqServer runs task handlers registered by modules.
// It wraps an asynq server and mux, dispatching registered task types to the
// handlers while asynq owns retries and concurrency.
type AsynqServer struct {
	srv *asynq.Server
	mux *asynq.ServeMux
}

// newAsynqServer builds an asynq server from config. The server logs through
// the provided slog logger, records failures via the error handler, and
// reports liveness through the health check.
func newAsynqServer(cfg config.QueueConfig, log *slog.Logger) (*AsynqServer, error) {
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
			// Never log the payload: OTP and magic-link material would leak
			// one-time secrets into the log stream.
			log.Error("task failed",
				"task_type", task.Type,
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
	return &AsynqServer{srv: srv, mux: asynq.NewServeMux()}, nil
}

// Register implements queue.Registrar.
// Each call binds one task type to a handler; a queue.ErrPermanent return
// value is translated into asynq.SkipRetry.
func (s *AsynqServer) Register(taskType string, h queue.TaskHandler) {
	s.mux.HandleFunc(taskType, func(ctx context.Context, t *asynq.Task) error {
		err := h(ctx, t.Payload())
		if errors.Is(err, queue.ErrPermanent) {
			return asynq.SkipRetry
		}
		return err
	})
}

// Run blocks and processes tasks until Stop is called.
// It returns an error wrapping asynq's when the server fails to run.
func (s *AsynqServer) Run() error {
	if err := s.srv.Run(s.mux); err != nil {
		return fmt.Errorf("asynq server: %w", err)
	}
	return nil
}

// Stop gracefully stops the server.
// It triggers asynq's shutdown, letting in-flight tasks drain.
func (s *AsynqServer) Stop() { s.srv.Shutdown() }

// slogAdapter adapts slog to asynq.Logger.
type slogAdapter struct {
	log *slog.Logger
}

// Debug logs at slog's debug level.
// The first argument is used as the message and the rest are ignored.
func (a slogAdapter) Debug(args ...any) { a.log.Debug(formatArgs(args...)) }

// Info logs at slog's info level.
// The first argument is used as the message and the rest are ignored.
func (a slogAdapter) Info(args ...any) { a.log.Info(formatArgs(args...)) }

// Warn logs at slog's warn level.
// The first argument is used as the message and the rest are ignored.
func (a slogAdapter) Warn(args ...any) { a.log.Warn(formatArgs(args...)) }

// Error logs at slog's error level.
// The first argument is used as the message and the rest are ignored.
func (a slogAdapter) Error(args ...any) { a.log.Error(formatArgs(args...)) }

// Fatal logs at slog's error level; asynq has no fatal state.
// The first argument is used as the message and the rest are ignored.
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
