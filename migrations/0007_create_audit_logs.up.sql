CREATE TABLE IF NOT EXISTS audit_logs (
    id           VARCHAR(64)   NOT NULL PRIMARY KEY,
    subject_type VARCHAR(191)  NOT NULL,
    subject_id   VARCHAR(64)   NOT NULL,
    action       VARCHAR(16)   NOT NULL,
    old_values   TEXT          NULL,
    new_values   TEXT          NULL,
    actor_type   VARCHAR(16)   NOT NULL DEFAULT 'system',
    actor_id     VARCHAR(64)   NULL,
    ip_address   VARCHAR(64)   NULL,
    user_agent   VARCHAR(512)  NULL,
    created_at   TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_subject ON audit_logs (subject_type, subject_id, created_at);
CREATE INDEX idx_audit_logs_actor ON audit_logs (actor_type, actor_id);
