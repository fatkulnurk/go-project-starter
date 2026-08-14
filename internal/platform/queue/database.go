package queue

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/id"
	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
)

const (
	defaultPollInterval = 1 * time.Second
	defaultLease        = 60 * time.Second
	backoffBase         = 5 * time.Second
	backoffMax          = 5 * time.Minute
)

// DatabaseClient enqueues tasks into the queue_jobs table. It reuses the
// application SQL pool and therefore does not own it.
type DatabaseClient struct {
	db     *sql.DB
	driver string
}

// NewDatabaseClient builds a database-backed enqueuer.
func NewDatabaseClient(db *sql.DB, driver string) *DatabaseClient {
	return &DatabaseClient{db: db, driver: driver}
}

// Enqueue implements queue.Enqueuer.
func (c *DatabaseClient) Enqueue(ctx context.Context, t queue.Task) error {
	maxAttempts := 1
	if t.MaxRetry > 0 {
		maxAttempts = t.MaxRetry + 1
	}
	const q = `INSERT INTO queue_jobs (id, queue, payload, max_attempts, attempts, reserved_at, available_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, NULL, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
	_, err := c.db.ExecContext(ctx, database.Rebind(q, c.driver),
		id.New(), t.Type, string(t.Payload), maxAttempts, time.Now().UTC().Unix())
	return err
}

// Close implements the client cleanup contract; the pool is owned elsewhere.
func (c *DatabaseClient) Close() error { return nil }

// DatabaseServer polls the queue_jobs table and processes jobs with a worker
// pool. Each job is claimed atomically via SELECT ... FOR UPDATE and released
// back with a backoff when the handler fails. A lease keeps jobs claimed by a
// crashed worker reclaimable.
type DatabaseServer struct {
	db          *sql.DB
	driver      string
	concurrency int
	log         *slog.Logger
	handlers    map[string]queue.TaskHandler

	mu   sync.RWMutex
	stop chan struct{}
	wg   sync.WaitGroup
}

// NewDatabaseServer builds a database-backed worker.
func NewDatabaseServer(db *sql.DB, driver string, concurrency int, log *slog.Logger) *DatabaseServer {
	if concurrency <= 0 {
		concurrency = 10
	}
	return &DatabaseServer{
		db:          db,
		driver:      driver,
		concurrency: concurrency,
		log:         log,
		handlers:    make(map[string]queue.TaskHandler),
	}
}

// Register implements queue.Registrar.
func (s *DatabaseServer) Register(taskType string, h queue.TaskHandler) {
	s.mu.Lock()
	s.handlers[taskType] = h
	s.mu.Unlock()
}

// Run blocks and processes jobs until Stop is called.
func (s *DatabaseServer) Run() error {
	s.stop = make(chan struct{})
	for i := 0; i < s.concurrency; i++ {
		s.wg.Add(1)
		go s.worker()
	}
	s.wg.Wait()
	return nil
}

// Stop gracefully stops the workers.
func (s *DatabaseServer) Stop() {
	if s.stop == nil {
		return
	}
	close(s.stop)
}

// worker claims and processes one job at a time until stopped.
func (s *DatabaseServer) worker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stop:
			return
		default:
		}
		if s.processOne(context.Background()) {
			continue
		}
		select {
		case <-s.stop:
			return
		case <-time.After(defaultPollInterval):
		}
	}
}

// runHandler invokes a registered handler and converts a panic into an error
// so a misbehaving task cannot kill the worker.
func (s *DatabaseServer) runHandler(h queue.TaskHandler, ctx context.Context, j job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("queue handler panic: %v", r)
		}
	}()
	return h(ctx, j.payload)
}

// processOne claims a single available job and dispatches it to the
// registered handler. It returns false when no job was available.
func (s *DatabaseServer) processOne(ctx context.Context) bool {
	job, ok, err := s.claim(ctx)
	if err != nil {
		s.log.Error("queue claim failed", "err", err)
		return false
	}
	if !ok {
		return false
	}

	s.mu.RLock()
	h, found := s.handlers[job.queue]
	s.mu.RUnlock()
	if !found {
		s.log.Error("queue task has no handler", "task_type", job.queue)
		s.finish(ctx, job.id)
		return true
	}

	err = s.runHandler(h, ctx, job)
	if err == nil {
		s.finish(ctx, job.id)
		return true
	}
	if errors.Is(err, queue.ErrPermanent) {
		s.log.Warn("queue task skipped (permanent)", "task_type", job.queue, "err", err)
		s.finish(ctx, job.id)
		return true
	}

	// job.attempts is the pre-increment value read at claim time; the running
	// attempt is attempts+1. Giving up happens when the current attempt equals
	// maxAttempts, i.e. the job has been executed that many times.
	attempt := job.attempts + 1
	if attempt >= job.maxAttempts {
		s.log.Error("queue task gave up after max attempts", "task_type", job.queue, "attempts", attempt, "err", err)
		s.finish(ctx, job.id)
		return true
	}
	s.release(ctx, job.id, attempt)
	s.log.Warn("queue task failed, scheduling retry", "task_type", job.queue, "attempts", attempt, "err", err)
	return true
}

// claim atomically reserves one available job.
func (s *DatabaseServer) claim(ctx context.Context) (job, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return job{}, false, err
	}
	defer tx.Rollback() //nolint:errcheck // rolled back on error paths

	now := time.Now().UTC()
	const selectQ = `SELECT id, queue, payload, attempts, max_attempts FROM queue_jobs
		WHERE available_at <= ? AND (reserved_at IS NULL OR reserved_at < ?)
		ORDER BY available_at, created_at LIMIT 1 FOR UPDATE`
	var j job
	var payload string
	err = tx.QueryRowContext(ctx, database.Rebind(selectQ, s.driver),
		now.Unix(), now.Add(-defaultLease).Unix()).Scan(&j.id, &j.queue, &payload, &j.attempts, &j.maxAttempts)
	if errors.Is(err, sql.ErrNoRows) {
		return job{}, false, nil
	}
	if err != nil {
		return job{}, false, err
	}
	j.payload = []byte(payload)

	const updateQ = `UPDATE queue_jobs SET reserved_at = ?, attempts = attempts + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	if _, err := tx.ExecContext(ctx, database.Rebind(updateQ, s.driver), now.Unix(), j.id); err != nil {
		return job{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return job{}, false, err
	}
	return j, true, nil
}

// release makes a failed job available again after a growing backoff.
func (s *DatabaseServer) release(ctx context.Context, id string, attempts int) {
	delay := backoff(attempts)
	const q = `UPDATE queue_jobs SET reserved_at = NULL, available_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := s.db.ExecContext(ctx, database.Rebind(q, s.driver), time.Now().UTC().Add(delay).Unix(), id)
	if err != nil {
		s.log.Error("queue release failed", "id", id, "err", err)
	}
}

// backoff returns the retry delay after `attempts` failures. It doubles from
// backoffBase up to backoffMax using a guarded loop so large attempt counts
// cannot overflow the shift and produce a negative duration.
func backoff(attempts int) time.Duration {
	delay := backoffBase
	for i := 1; i < attempts; i++ {
		if delay >= backoffMax/2 {
			delay = backoffMax
			break
		}
		delay *= 2
	}
	return delay
}

// finish removes a processed or dropped job.
func (s *DatabaseServer) finish(ctx context.Context, id string) {
	const q = `DELETE FROM queue_jobs WHERE id = ?`
	_, err := s.db.ExecContext(ctx, database.Rebind(q, s.driver), id)
	if err != nil {
		s.log.Error("queue finish failed", "id", id, "err", err)
	}
}

// job is a claimed queue row.
type job struct {
	id          string
	queue       string
	payload     []byte
	attempts    int
	maxAttempts int
}

var _ queue.Enqueuer = (*DatabaseClient)(nil)
var _ Worker = (*DatabaseServer)(nil)
