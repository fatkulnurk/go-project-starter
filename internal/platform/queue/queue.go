// Package queue provides the queue backends and their factories. Business
// modules must only use the internal/application/queue contracts.
package queue

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/hibiken/asynq"
)

// Client is the union of the application enqueuer contract and cleanup.
type Client interface {
	queue.Enqueuer
	Close() error
}

// Worker is the union of task registration and the run/stop lifecycle.
type Worker interface {
	queue.Registrar
	Run() error
	Stop()
}

// NewClient builds an enqueuing client for the configured driver. db and
// dbDriver are only used when the driver is "db" and may be nil otherwise.
func NewClient(cfg config.QueueConfig, db *sql.DB, dbDriver string) (Client, error) {
	switch cfg.Driver {
	case config.DriverDB:
		if db == nil {
			return nil, fmt.Errorf("queue driver %q requires a database connection", cfg.Driver)
		}
		return NewDatabaseClient(db, dbDriver), nil
	case config.DriverAsynq:
		return newAsynqClient(cfg)
	default:
		return nil, fmt.Errorf("unknown queue driver %q", cfg.Driver)
	}
}

// NewServer builds a task-processing server for the configured driver. db and
// dbDriver are only used when the driver is "db" and may be nil otherwise.
func NewServer(cfg config.QueueConfig, log *slog.Logger, db *sql.DB, dbDriver string) (Worker, error) {
	switch cfg.Driver {
	case config.DriverDB:
		if db == nil {
			return nil, fmt.Errorf("queue driver %q requires a database connection", cfg.Driver)
		}
		return NewDatabaseServer(db, dbDriver, cfg.Concurrency, log), nil
	case config.DriverAsynq:
		return newAsynqServer(cfg, log)
	default:
		return nil, fmt.Errorf("unknown queue driver %q", cfg.Driver)
	}
}

// AsynqClient enqueues tasks to Redis via asynq.
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

// Enqueue implements queue.Enqueuer.
func (c *AsynqClient) Enqueue(ctx context.Context, t queue.Task) error {
	opts := make([]asynq.Option, 0, 1)
	if t.MaxRetry > 0 {
		opts = append(opts, asynq.MaxRetry(t.MaxRetry))
	}
	task := asynq.NewTask(t.Type, t.Payload, opts...)
	_, err := c.c.EnqueueContext(ctx, task)
	return err
}

// Close releases the client connection.
func (c *AsynqClient) Close() error { return c.c.Close() }

// AsynqServer runs task handlers registered by modules.
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
	return &AsynqServer{srv: srv, mux: asynq.NewServeMux()}, nil
}

// Register implements queue.Registrar.
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
func (s *AsynqServer) Run() error {
	if err := s.srv.Run(s.mux); err != nil {
		return fmt.Errorf("asynq server: %w", err)
	}
	return nil
}

// Stop gracefully stops the server.
func (s *AsynqServer) Stop() { s.srv.Shutdown() }

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
