-- =============================================================================
-- TABLE: verification_codes
-- One-time verification codes (OTP) and magic links. Used for:
--   - email / phone verification
--   - forgot password
--   - passwordless login (magic link)
-- The raw code is NOT stored, only its hash. A user may have several codes;
-- queries fetch the newest still-valid one (not consumed, not expired).
-- attempts counts failed submissions for brute-force protection.
--
-- channel values: 'email' | 'sms'
-- purpose values: 'verify_email' | 'verify_phone' | 'reset_password' | 'magic_link'
--
-- Example data:
--   id          = '0195c5d6-4b2e-7d00-8000-000000000001'  (UUID v7)
--   user_id     = '0195c5d4-2b40-7d00-8000-000000000001'  (FK -> users.id)
--   channel     = 'email'
--   purpose     = 'verify_email'
--   code_hash   = sha256 hex of the OTP (6 digits) or magic link token
--   attempts    = 0                   (increments on each wrong input)
--   expires_at  = '2026-01-15 10:45:00' (default TTL 15m)
--   consumed_at = NULL                (NULL = still valid; set on success)
--   created_at  = '2026-01-15 10:30:00'
--   updated_at  = '2026-01-15 10:30:00' (updated on consume/attempts)
-- =============================================================================
CREATE TABLE IF NOT EXISTS verification_codes (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (generated in app)
    user_id     VARCHAR(36)  NOT NULL,             -- code owner (FK users.id)
    channel     VARCHAR(16)  NOT NULL,             -- 'email' | 'sms'
    purpose     VARCHAR(24)  NOT NULL,             -- kind of code
    code_hash   VARCHAR(64)  NOT NULL,             -- SHA-256 of code/magic-link token
    attempts    INT          NOT NULL DEFAULT 0,   -- failed attempt count
    expires_at  TIMESTAMP    NOT NULL,             -- expiry time
    consumed_at TIMESTAMP    NULL,                 -- when the code was used (NULL = active)
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_verification_codes_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_verification_codes_user ON verification_codes (user_id, purpose, created_at);
