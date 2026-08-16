// Package queue defines the cross-cutting queue contract. Business modules
// only enqueue tasks and register handlers; the concrete backend (asynq,
// others) is hidden behind Enqueuer and Registrar.
package queue

import (
	"context"
	"errors"
)

// ErrPermanent signals a corrupt/unprocessable payload: do not retry.
var ErrPermanent = errors.New("permanent failure, skip retry")

// Task is a unit of work to be processed asynchronously. Type selects the
// registered handler; Payload is the opaque job data; MaxRetry is the number
// of times the backend may re-attempt a failed job (0 = no retries).
type Task struct {
	Type     string
	Payload  []byte
	MaxRetry int
}

// Enqueuer pushes tasks onto the queue.
// Implementations (asynq, database) are chosen in the composition root; the
// contract hides the backend entirely from business modules.
type Enqueuer interface {
	// Enqueue submits a task for asynchronous processing. It returns an error
	// when the backend rejects the task (e.g. Redis unavailable).
	Enqueue(ctx context.Context, t Task) error
}

// TaskHandler processes a single task payload. Returning ErrPermanent skips
// retries; any other error is retried by the backend up to the task's MaxRetry.
type TaskHandler func(ctx context.Context, payload []byte) error

// Registrar registers task handlers on a worker.
// A worker calls Register once per task type before Run starts processing.
type Registrar interface {
	// Register binds a task type to its handler. Registering the same type
	// twice replaces the previous handler.
	Register(taskType string, h TaskHandler)
}
