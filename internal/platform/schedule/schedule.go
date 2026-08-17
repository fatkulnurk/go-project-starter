// Package schedule provides the scheduled-jobs backend behind the
// internal/application/schedule contract. Business modules only register jobs;
// the concrete scheduling loop (stdlib time.Ticker) is hidden behind Worker.
package schedule

import (
	"context"
	"log/slog"
	"sync"
	"time"

	appschedule "github.com/fatkulnurk/go-project-starter/internal/application/schedule"
)

// Worker is the union of job registration and the run/stop lifecycle.
// The composition root hands it to modules so they can register jobs before
// Run starts executing them.
type Worker interface {
	appschedule.Registrar

	// Run blocks and executes registered jobs until Stop is called. It returns
	// after every in-flight handler has finished (their contexts are
	// cancelled by Stop).
	Run() error

	// Stop gracefully stops the scheduler: it cancels the contexts of running
	// handlers and waits for Run to return.
	Stop()
}

// New builds a stdlib time.Ticker backed scheduler. log receives handler
// failures and invalid registrations.
func New(log *slog.Logger) Worker {
	if log == nil {
		log = slog.Default()
	}
	return &tickerWorker{log: log, jobs: make(map[string]appschedule.Job), done: make(chan struct{})}
}

// tickerWorker runs one goroutine per registered job, ticking at the job's
// Interval until the scheduler stops.
type tickerWorker struct {
	log      *slog.Logger
	mu       sync.Mutex
	jobs     map[string]appschedule.Job
	done     chan struct{}
	stopOnce sync.Once
}

// Register implements schedule.Registrar. A non-positive interval disables the
// job with a logged error. Registering the same name twice replaces the
// previous job's schedule.
func (w *tickerWorker) Register(j appschedule.Job) {
	if j.Interval <= 0 {
		w.log.Error("invalid schedule interval, job disabled", "job", j.Name, "interval", j.Interval.String())
		return
	}
	w.mu.Lock()
	w.jobs[j.Name] = j
	w.mu.Unlock()
}

// Run implements Worker. It snapshots the registered jobs, starts a goroutine
// per job, then blocks until Stop cancels the shared context. Each handler
// runs with a cancellable context so a graceful shutdown aborts in-flight work.
func (w *tickerWorker) Run() error {
	w.mu.Lock()
	jobs := make([]appschedule.Job, 0, len(w.jobs))
	for _, j := range w.jobs {
		jobs = append(jobs, j)
	}
	w.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(j appschedule.Job) {
			defer wg.Done()
			w.tick(j, ctx)
		}(j)
	}

	<-w.done
	cancel()
	wg.Wait()
	return nil
}

// tick runs the job handler once per Interval until ctx is cancelled.
// Handler errors are logged, not propagated, so one failing job never stops
// the scheduler or its siblings.
func (w *tickerWorker) tick(j appschedule.Job, ctx context.Context) {
	t := time.NewTicker(j.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := j.Handler(ctx); err != nil {
				w.log.Error("scheduled job failed", "job", j.Name, "err", err)
			}
		}
	}
}

// Stop implements Worker. It is safe to call multiple times and from any
// goroutine; the first call unblocks Run.
func (w *tickerWorker) Stop() {
	w.stopOnce.Do(func() {
		close(w.done)
	})
}
