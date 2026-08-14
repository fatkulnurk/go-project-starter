-- =============================================================================
-- TABLE: queue_jobs
-- Backing store untuk queue driver berbasis database (QUEUE_DRIVER=db). Skema
-- mengikuti Laravel database queue: satu baris = satu task. queue berisi jenis
-- task (sama dengan Task.Type, contoh 'auth.send_email_verification'), payload
-- berisi JSON data task, max_attempts = Task.MaxRetry + 1 (batas jumlah klaim).
-- Worker mengklaim baris dengan mengunci lewat SELECT ... FOR UPDATE:
-- reserved_at diisi saat diklaim (lease), attempts bertambah tiap percobaan,
-- available_at menentukan kapan task boleh dieksekusi (dipakai untuk
-- delay/retry bertahap).
--
-- Contoh data:
--   id           = '0195c5d9-a033-7d00-8000-000000000001'  (UUID v7)
--   queue        = 'auth.send_email_verification'
--   payload      = '{"to":"budi@example.com","code":"123456"}'
--   max_attempts = 4                       (MaxRetry=3, jadi maks 4 klaim)
--   attempts     = 2
--   reserved_at  = 1768449600                (NULL = tidak sedang diklaim)
--   available_at = 1768449605                (retry backoff +5 detik)
--   created_at   = '2026-01-15 10:30:00'
--   updated_at   = '2026-01-15 10:30:00'
-- =============================================================================
CREATE TABLE IF NOT EXISTS queue_jobs (
    id           VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (dibuat di app)
    queue        VARCHAR(191) NOT NULL,             -- jenis task (Task.Type)
    payload      TEXT         NOT NULL,             -- JSON data task
    max_attempts INT          NOT NULL DEFAULT 1,   -- batas jumlah percobaan
    attempts     INT          NOT NULL DEFAULT 0,   -- jumlah percobaan berjalan
    reserved_at  BIGINT       NULL,                 -- unix detik saat diklaim (lease)
    available_at BIGINT       NOT NULL DEFAULT 0,   -- unix detik kapan boleh dieksekusi
    created_at   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_queue_jobs_available ON queue_jobs (available_at);