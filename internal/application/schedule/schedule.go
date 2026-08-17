// Package schedule defines the cross-cutting scheduled-jobs contract. Business
// modules register periodic jobs and the concrete backend (ticker, cron,
// others) is hidden behind Registrar.
package schedule

import (
	"context"
	"time"
)

// Job is a unit of periodic work. Name identifies the job in logs; Interval
// is how often Handler runs; Handler performs the actual work.
type Job struct {
	Name     string
	Interval time.Duration
	Handler  JobHandler
}

// JobHandler runs a single execution of a job. The context is cancelled when
// the scheduler stops, so a long-running handler can abort early. Returning an
// error is logged by the scheduler and does not stop the job.
type JobHandler func(ctx context.Context) error

// Registrar registers periodic jobs on a scheduler.
// A scheduler starts executing registered jobs after Run; registering the same
// name twice replaces the previous job.
type Registrar interface {
	// Register adds a job to the scheduler.
	Register(j Job)
}
