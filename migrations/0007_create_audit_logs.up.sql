-- =============================================================================
-- TABLE: audit_logs
-- Global audit trail (immutable log) of important user/admin actions: who did
-- what, to which object, and how the values changed before/after. One row = one
-- event. old_values/new_values are JSON strings (e.g. '{"email":"a@b.c"}')
-- holding the changed columns. actor_type = 'system' (middleware, no logged-in
-- user) or 'user'. Used for forensics, compliance, and tracking suspicious
-- activity.
--
-- action values : 'create' | 'update' | 'delete' | 'login' | 'logout' |
--                 'register' | 'verify' | 'reset_password' | 'revoke' | ...
--
-- Example data:
--   id           = '0195c5da-b134-7d00-8000-000000000001'  (UUID v7)
--   subject_type = 'user'                (entity that changed)
--   subject_id   = '0195c5d4-2b40-7d00-8000-000000000001'  (id of the changed entity)
--   action       = 'update'
--   old_values   = '{"name":"Budi Lama"}'
--   new_values   = '{"name":"Budi Baru"}'
--   actor_type   = 'user' | 'system'
--   actor_id     = '0195c5d4-2b40-7d00-8000-000000000001'  (acting user; NULL if system)
--   ip_address   = '192.168.1.10'
--   user_agent   = 'Mozilla/5.0 ...'
--   created_at   = '2026-01-15 10:30:00'
--   updated_at   = '2026-01-15 10:30:00' (same as created_at, logs are immutable)
-- =============================================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id           VARCHAR(36)   NOT NULL PRIMARY KEY, -- UUID v7 (generated in app)
    subject_type VARCHAR(191)  NOT NULL,             -- entity type (e.g. 'user', 'role')
    subject_id   VARCHAR(36)   NOT NULL,             -- id of the changed entity
    action       VARCHAR(16)   NOT NULL,             -- action kind (create/update/...)
    old_values   TEXT          NULL,                 -- JSON of the value before the change
    new_values   TEXT          NULL,                 -- JSON of the value after the change
    actor_type   VARCHAR(16)   NOT NULL DEFAULT 'system', -- 'system' | 'user'
    actor_id     VARCHAR(36)   NULL,                 -- actor id (NULL = system)
    ip_address   VARCHAR(64)   NULL,                 -- request source IP
    user_agent   VARCHAR(512)  NULL,                 -- browser/client user-agent
    created_at   TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_subject ON audit_logs (subject_type, subject_id, created_at);
CREATE INDEX idx_audit_logs_actor ON audit_logs (actor_type, actor_id);
