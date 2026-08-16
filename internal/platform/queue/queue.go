// Package queue provides the queue backends and their factories. Business
// modules must only use the internal/application/queue contracts.
package queue

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// Client is the union of the application enqueuer contract and cleanup.
// Modules receive this from the composition root to dispatch background work
// without knowing which backend is configured.
type Client interface {
	queue.Enqueuer

	// Close releases the backend connection. For database-backed clients it
	// is a no-op because the shared SQL pool is owned elsewhere.
	Close() error
}

// Worker is the union of task registration and the run/stop lifecycle.
// The composition root hands it to modules so they can register handlers
// before Run starts processing tasks.
type Worker interface {
	queue.Registrar

	// Run blocks and processes tasks until Stop is called. It returns an
	// error when the underlying backend fails to start.
	Run() error

	// Stop gracefully stops processing tasks. For the database backend it
	// also cancels in-flight handler contexts.
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
