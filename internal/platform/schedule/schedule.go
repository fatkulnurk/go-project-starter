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

// New builds a stdlib-backed cron scheduler. loc is the timezone in which
// cron expressions are evaluated (pass cfg.Location()); log receives handler
// failures and invalid registrations.
func New(log *slog.Logger, loc *time.Location) Worker {
	if log == nil {
		log = slog.Default()
	}
	if loc == nil {
		loc = time.UTC
	}
	return &cronWorker{log: log, loc: loc, jobs: make(map[string]registeredJob), done: make(chan struct{})}
}

// registeredJob pairs a registered job with its compiled cron expression.
type registeredJob struct {
	appschedule.Job
	spec *Spec
}

// cronWorker runs one goroutine per registered job, waking at every minute
// boundary (in its location) and running the handler when the cron expression
// matches, until the scheduler stops.
type cronWorker struct {
	log      *slog.Logger
	loc      *time.Location
	mu       sync.Mutex
	jobs     map[string]registeredJob
	done     chan struct{}
	stopOnce sync.Once
}

// Register implements schedule.Registrar. An invalid cron expression disables
// the job with a logged error. Registering the same name twice replaces the
// previous job.
func (w *cronWorker) Register(j appschedule.Job) {
	spec, err := Parse(j.Cron)
	if err != nil {
		w.log.Error("invalid cron expression, job disabled", "job", j.Name, "cron", j.Cron, "err", err)
		return
	}
	w.mu.Lock()
	w.jobs[j.Name] = registeredJob{Job: j, spec: spec}
	w.mu.Unlock()
}

// Run implements Worker. It snapshots the registered jobs, starts a goroutine
// per job, then blocks until Stop cancels the shared context. Each handler
// runs with a cancellable context so a graceful shutdown aborts in-flight work.
func (w *cronWorker) Run() error {
	w.mu.Lock()
	jobs := make([]registeredJob, 0, len(w.jobs))
	for _, j := range w.jobs {
		jobs = append(jobs, j)
	}
	w.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(j registeredJob) {
			defer wg.Done()
			w.loop(ctx, j)
		}(j)
	}

	<-w.done
	cancel()
	wg.Wait()
	return nil
}

// loop wakes at the start of every minute in w.loc and runs the handler when
// the job's cron expression matches that minute.
func (w *cronWorker) loop(ctx context.Context, j registeredJob) {
	for {
		if !w.sleepUntilNextMinute(ctx) {
			return
		}
		now := time.Now().In(w.loc)
		if j.spec.Match(now) {
			if err := j.Handler(ctx); err != nil {
				w.log.Error("scheduled job failed", "job", j.Name, "err", err)
			}
		}
	}
}

// sleepUntilNextMinute sleeps until the start of the next minute in w.loc. It
// returns false if ctx is cancelled before the timer fires.
func (w *cronWorker) sleepUntilNextMinute(ctx context.Context) bool {
	now := time.Now().In(w.loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, w.loc).Add(time.Minute)

	timer := time.NewTimer(time.Until(next))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// Stop implements Worker. It is safe to call multiple times and from any
// goroutine; the first call unblocks Run.
func (w *cronWorker) Stop() {
	w.stopOnce.Do(func() {
		close(w.done)
	})
}
