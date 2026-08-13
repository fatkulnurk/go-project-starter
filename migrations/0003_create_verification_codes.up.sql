CREATE TABLE IF NOT EXISTS verification_codes (
    id          VARCHAR(64)  NOT NULL PRIMARY KEY,
    user_id     VARCHAR(64)  NOT NULL,
    channel     VARCHAR(16)  NOT NULL,
    purpose     VARCHAR(24)  NOT NULL,
    code_hash   VARCHAR(64)  NOT NULL,
    attempts    INT          NOT NULL DEFAULT 0,
    expires_at  TIMESTAMP    NOT NULL,
    consumed_at TIMESTAMP    NULL,
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_verification_codes_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_verification_codes_user ON verification_codes (user_id, purpose, created_at);
