-- =============================================================================
-- TABLE: users
-- Application user accounts. One row = one user (email and/or phone number).
-- Status marks the account active or suspended; the *verified_at columns mark
-- whether the email/phone has been OTP-verified. totp_secret is the base32 MFA
-- shared secret and totp_confirmed_at marks the secret as active (a staged but
-- unconfirmed secret must never be enforced at login). All id columns contain
-- application-generated UUID v7 (36 chars), never database-generated.
--
-- Example data:
--   id                = '0195c5d4-2b40-7d00-8000-000000000001'  (UUID v7)
--   name              = 'Budi Santoso'
--   email             = 'budi@example.com'
--   phone             = '+6281234567890'       (NULL if none)
--   password_hash     = '$2a$10$...'           (bcrypt/argon2 hash)
--   email_verified_at = '2026-01-15 10:30:00'  (NULL = not verified)
--   phone_verified_at = NULL
--   status            = 'active' | 'suspended'
--   created_at        = '2026-01-15 10:30:00'
--   updated_at        = '2026-01-15 10:30:00'
-- =============================================================================
CREATE TABLE IF NOT EXISTS users (
    id                VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (generated in app)
    name              VARCHAR(255) NOT NULL,             -- user display name
    email             VARCHAR(255) NULL,                 -- login email (unique, may be empty if using phone)
    phone             VARCHAR(32)  NULL,                 -- login phone (unique, may be empty if using email)
    password_hash     VARCHAR(255) NOT NULL,             -- password hash, never plaintext
    email_verified_at TIMESTAMP    NULL,                 -- when the email was verified (NULL = not yet)
    phone_verified_at TIMESTAMP    NULL,                 -- when the phone was verified (NULL = not yet)
    totp_secret       VARCHAR(64)  NOT NULL DEFAULT '',  -- base32 MFA secret ('' = disabled)
    totp_confirmed_at TIMESTAMP    NULL,                 -- when MFA was activated (NULL = not active)
    status            VARCHAR(20)  NOT NULL DEFAULT 'active', -- 'active' | 'suspended'
    created_at        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (email),
    UNIQUE (phone)
);

-- =============================================================================
-- TABLE: user_recovery_codes
-- One-time MFA fallback codes shown to the user when MFA is activated. Only
-- their SHA-256 hash is stored (64 hex chars), so a database leak does not
-- leak usable codes; a used code is marked instead of deleted to stay auditable.
-- =============================================================================
CREATE TABLE IF NOT EXISTS user_recovery_codes (
    user_id    VARCHAR(36) NOT NULL,             -- owner (FK users.id)
    code_hash  VARCHAR(64) NOT NULL,             -- SHA-256 of the raw code
    used_at    TIMESTAMP   NULL,                 -- when the code was consumed (NULL = unused)
    created_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, code_hash),
    CONSTRAINT fk_recovery_codes_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
