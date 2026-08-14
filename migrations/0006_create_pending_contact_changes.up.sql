-- =============================================================================
-- TABLE: pending_contact_changes
-- Perubahan kontak (email/HP) yang belum tuntas. Flow-nya: user mengganti email
-- -> baris pending dibuat dengan status 'pending' -> OTP dikirim ke alamat baru
-- -> saat OTP diverifikasi, perubahan diterapkan ke users + baris ini menjadi
-- 'applied'. Tujuan: tidak langsung mengubah email/HP utama sebelum dibuktikan
-- bahwa alamat baru itu benar-benar milik user (anti-hijack akun).
--
-- Nilai channel : 'email' | 'sms'
-- Nilai status  : 'pending' | 'applied'
--
-- Contoh data:
--   id         = '0195c5d9-a033-7d00-8000-000000000001'  (UUID v7)
--   user_id    = '0195c5d4-2b40-7d00-8000-000000000001'  (FK -> users.id)
--   channel    = 'email'
--   old_value  = 'budi@example.com'     (nilai lama yang akan diganti)
--   new_value  = 'budi.baru@example.com'(nilai baru tujuan)
--   status     = 'pending'
--   applied_at = NULL                   (terisi setelah diverifikasi)
--   created_at = '2026-01-15 10:30:00'
--   updated_at = '2026-01-15 10:30:00' (ter-update saat applied)
-- =============================================================================
CREATE TABLE IF NOT EXISTS pending_contact_changes (
    id          VARCHAR(36)   NOT NULL PRIMARY KEY, -- UUID v7 (dibuat di app)
    user_id     VARCHAR(36)   NOT NULL,             -- pemilik akun (FK users.id)
    channel     VARCHAR(16)   NOT NULL,             -- 'email' | 'sms'
    old_value   VARCHAR(255)  NOT NULL,             -- kontak lama
    new_value   VARCHAR(255)  NOT NULL,             -- kontak baru (tujuan)
    status      VARCHAR(16)   NOT NULL DEFAULT 'pending', -- 'pending' | 'applied'
    applied_at  TIMESTAMP     NULL,                 -- waktu perubahan diterapkan
    created_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_pending_contact_changes_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_pending_contact_changes_user ON pending_contact_changes (user_id, channel, status, created_at);
CREATE INDEX idx_pending_contact_changes_new ON pending_contact_changes (channel, new_value, status);
