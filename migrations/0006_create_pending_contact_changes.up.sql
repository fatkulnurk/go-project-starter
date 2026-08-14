-- =============================================================================
-- TABLE: pending_contact_changes
-- Contact (email/phone) changes that are not yet complete. Flow: a user changes
-- their email -> a pending row is created with status 'pending' -> an OTP is
-- sent to the new address -> when the OTP is verified the change is applied to
-- users and this row becomes 'applied'. Purpose: never change the primary
-- email/phone before proving the new address really belongs to the user
-- (anti account hijacking).
--
-- channel values: 'email' | 'sms'
-- status values : 'pending' | 'applied'
--
-- Example data:
--   id         = '0195c5d9-a033-7d00-8000-000000000001'  (UUID v7)
--   user_id    = '0195c5d4-2b40-7d00-8000-000000000001'  (FK -> users.id)
--   channel    = 'email'
--   old_value  = 'budi@example.com'     (old value being replaced)
--   new_value  = 'budi.baru@example.com'(new target value)
--   status     = 'pending'
--   applied_at = NULL                   (filled after verification)
--   created_at = '2026-01-15 10:30:00'
--   updated_at = '2026-01-15 10:30:00' (updated on applied)
-- =============================================================================
CREATE TABLE IF NOT EXISTS pending_contact_changes (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY, -- UUID v7 (generated in app)
    user_id     VARCHAR(36)   NOT NULL,             -- account owner (FK users.id)
    channel     VARCHAR(16)   NOT NULL,             -- 'email' | 'sms'
    old_value   VARCHAR(255)  NOT NULL,             -- old contact
    new_value   VARCHAR(255)  NOT NULL,             -- new contact (target)
    status      VARCHAR(16)   NOT NULL DEFAULT 'pending', -- 'pending' | 'applied'
    applied_at  TIMESTAMP     NULL,                 -- when the change was applied
    created_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_pending_contact_changes_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_pending_contact_changes_user ON pending_contact_changes (user_id, channel, status, created_at);
CREATE INDEX idx_pending_contact_changes_new ON pending_contact_changes (channel, new_value, status);
