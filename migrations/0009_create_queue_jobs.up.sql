-- =============================================================================
-- TABLE: queue_jobs
-- Backing store for the database-based queue driver (QUEUE_DRIVER=db). The
-- schema follows Laravel's database queue: one row = one task. queue holds the
-- task kind (same as Task.Type, e.g. 'auth.send_email_verification'), payload
-- holds the JSON task data, max_attempts = Task.MaxRetry + 1 (claim limit).
-- The worker claims a row by locking via SELECT ... FOR UPDATE: reserved_at is
-- set when claimed (lease), attempts increments per try, available_at decides
-- when the task may run (used for delay / staggered retries).
--
-- Example data:
--   id           = '0195c5d9-a033-7d00-8000-000000000001'  (UUID v7)
--   queue        = 'auth.send_email_verification'
--   payload      = '{"to":"budi@example.com","code":"123456"}'
--   max_attempts = 4                       (MaxRetry=3, so max 4 claims)
--   attempts     = 2
--   reserved_at  = 1768449600                (NULL = not currently claimed)
--   available_at = 1768449605                (retry backoff +5 seconds)
--   created_at   = '2026-01-15 10:30:00'
--   updated_at   = '2026-01-15 10:30:00'
-- =============================================================================
CREATE TABLE IF NOT EXISTS queue_jobs (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (generated in app)
    queue        VARCHAR(191) NOT NULL,             -- task kind (Task.Type)
    payload      TEXT         NOT NULL,             -- JSON task data
    max_attempts INT          NOT NULL DEFAULT 1,   -- claim limit
    attempts     INT          NOT NULL DEFAULT 0,   -- running attempt count
    reserved_at  BIGINT       NULL,                 -- unix seconds when claimed (lease)
    available_at BIGINT       NOT NULL DEFAULT 0,   -- unix seconds when eligible to run
    created_at   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_queue_jobs_available ON queue_jobs (available_at);