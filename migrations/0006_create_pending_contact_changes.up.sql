CREATE TABLE IF NOT EXISTS pending_contact_changes (
    id          VARCHAR(64)   NOT NULL PRIMARY KEY,
    user_id     VARCHAR(64)   NOT NULL,
    channel     VARCHAR(16)   NOT NULL,
    old_value   VARCHAR(255)  NOT NULL,
    new_value   VARCHAR(255)  NOT NULL,
    status      VARCHAR(16)   NOT NULL DEFAULT 'pending',
    applied_at  TIMESTAMP     NULL,
    created_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_pending_contact_changes_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_pending_contact_changes_user ON pending_contact_changes (user_id, channel, status, created_at);
CREATE INDEX idx_pending_contact_changes_new ON pending_contact_changes (channel, new_value, status);
