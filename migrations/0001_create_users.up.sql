-- =============================================================================
-- TABLE: users
-- Application user accounts. One row = one user (email and/or phone number).
-- Status marks the account active or suspended; the *verified_at columns mark
-- whether the email/phone has been OTP-verified. All id columns contain
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
    status            VARCHAR(20)  NOT NULL DEFAULT 'active', -- 'active' | 'suspended'
    created_at        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (email),
    UNIQUE (phone)
);
