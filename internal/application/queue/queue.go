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

// Task is a unit of work to be processed asynchronously.
type Task struct {
	Type     string
	Payload  []byte
	MaxRetry int
}

// Enqueuer pushes tasks onto the queue.
type Enqueuer interface {
	Enqueue(ctx context.Context, t Task) error
}

// TaskHandler processes a single task payload. Returning ErrPermanent skips
// retries; any other error is retried by the backend.
type TaskHandler func(ctx context.Context, payload []byte) error

// Registrar registers task handlers on a worker.
type Registrar interface {
	Register(taskType string, h TaskHandler)
}
