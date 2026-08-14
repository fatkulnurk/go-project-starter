-- =============================================================================
-- TABLE: audit_logs
-- Jejak audit global (immutable log) dari aksi penting pengguna/admin: siapa
-- melakukan apa, pada objek apa, sebelum/sesudah nilainya bagaimana. Satu baris
-- = satu kejadian. old_values/new_values berupa JSON string (mis.
-- '{"email":"a@b.c"}' ) berisi kolom yang berubah. actor_type = 'system'
-- (middleware, tidak ada user login) atau 'user'. Dipakai untuk forensik,
-- compliance, dan melacak aktivitas mencurigakan.
--
-- Nilai action   : 'create' | 'update' | 'delete' | 'login' | 'logout' |
--                  'register' | 'verify' | 'reset_password' | 'revoke' | ...
--
-- Contoh data:
--   id           = '0195c5da-b134-7d00-8000-000000000001'  (UUID v7)
--   subject_type = 'user'                (entitas yang diubah)
--   subject_id   = '0195c5d4-2b40-7d00-8000-000000000001'  (id entitas yang diubah)
--   action       = 'update'
--   old_values   = '{"name":"Budi Lama"}'
--   new_values   = '{"name":"Budi Baru"}'
--   actor_type   = 'user' | 'system'
--   actor_id     = '0195c5d4-2b40-7d00-8000-000000000001'  (user pelaku; NULL jika system)
--   ip_address   = '192.168.1.10'
--   user_agent   = 'Mozilla/5.0 ...'
--   created_at   = '2026-01-15 10:30:00'
--   updated_at   = '2026-01-15 10:30:00' (sama dengan created_at, log tidak diubah)
-- =============================================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id           VARCHAR(36)   NOT NULL PRIMARY KEY, -- UUID v7 (dibuat di app)
    subject_type VARCHAR(191)  NOT NULL,             -- jenis entitas (mis. 'user', 'role')
    subject_id   VARCHAR(36)   NOT NULL,             -- id entitas yang diubah
    action       VARCHAR(16)   NOT NULL,             -- jenis aksi (create/update/...)
    old_values   TEXT          NULL,                 -- JSON nilai sebelum berubah
    new_values   TEXT          NULL,                 -- JSON nilai sesudah berubah
    actor_type   VARCHAR(16)   NOT NULL DEFAULT 'system', -- 'system' | 'user'
    actor_id     VARCHAR(36)   NULL,                 -- id pelaku (NULL = system)
    ip_address   VARCHAR(64)   NULL,                 -- IP asal request
    user_agent   VARCHAR(512)  NULL,                 -- user-agent browser/client
    created_at   TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_subject ON audit_logs (subject_type, subject_id, created_at);
CREATE INDEX idx_audit_logs_actor ON audit_logs (actor_type, actor_id);
