-- =============================================================================
-- TABLE: refresh_tokens
-- Long-lived login sessions (refresh tokens). Access JWTs are short-lived
-- (15m); when one expires the client exchanges its refresh token here for a new
-- access token. The raw token is NOT stored; only its SHA-256 hash is, so a
-- database leak does not leak usable tokens. Logout/revoke marks revoked_at
-- instead of deleting the row (so it stays auditable).
--
-- Example data:
--   id         = '0195c5d5-3a1f-7d00-8000-000000000001'  (UUID v7)
--   user_id    = '0195c5d4-2b40-7d00-8000-000000000001'  (FK -> users.id)
--   token_hash = sha256 hex of the raw token (64 chars)
--   expires_at = '2026-02-15 10:30:00'     (default TTL 720h / 30 days)
--   revoked_at = NULL                       (NULL = still active)
--   created_at = '2026-01-15 10:30:00'
--   updated_at = '2026-01-15 10:30:00'     (updated on revoke)
-- =============================================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (generated in app)
    user_id    VARCHAR(36)  NOT NULL,             -- token owner (FK users.id)
    token_hash VARCHAR(64)  NOT NULL,             -- SHA-256 of raw token (unique)
    expires_at TIMESTAMP    NOT NULL,             -- expiry time
    revoked_at TIMESTAMP    NULL,                 -- when revoked/logged out (NULL = active)
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (token_hash),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id);
