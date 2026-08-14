-- =============================================================================
-- TABLE: verification_codes
-- Kode verifikasi sekali pakai (OTP) dan magic link. Dipakai untuk:
--   - verifikasi email / nomor HP
--   - forgot password
--   - login passwordless (magic link)
-- Kode asli TIDAK disimpan; hanya hash-nya. Satu user boleh punya beberapa
-- kode; query mengambil yang terbaru yang masih berlaku (belum consumed, belum
-- expired). attempts mencatat berapa kali gagal memasukkan kode untuk anti-brute-force.
--
-- Nilai channel  : 'email' | 'sms'
-- Nilai purpose  : 'verify_email' | 'verify_phone' | 'reset_password' | 'magic_link'
--
-- Contoh data:
--   id          = '0195c5d6-4b2e-7d00-8000-000000000001'  (UUID v7)
--   user_id     = '0195c5d4-2b40-7d00-8000-000000000001'  (FK -> users.id)
--   channel     = 'email'
--   purpose     = 'verify_email'
--   code_hash   = sha256 hex dari kode OTP (6 digit) atau magic link token
--   attempts    = 0                   (bertambah setiap salah input)
--   expires_at  = '2026-01-15 10:45:00' (TTL default 15m)
--   consumed_at = NULL                (NULL = masih berlaku; terisi saat sukses)
--   created_at  = '2026-01-15 10:30:00'
--   updated_at  = '2026-01-15 10:30:00' (ter-update saat consumed/attempts)
-- =============================================================================
CREATE TABLE IF NOT EXISTS verification_codes (
    id          VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (dibuat di app)
    user_id     VARCHAR(36)  NOT NULL,             -- pemilik kode (FK users.id)
    channel     VARCHAR(16)  NOT NULL,             -- 'email' | 'sms'
    purpose     VARCHAR(24)  NOT NULL,             -- jenis keperluan kode
    code_hash   VARCHAR(64)  NOT NULL,             -- SHA-256 kode/magic-link token
    attempts    INT          NOT NULL DEFAULT 0,   -- jumlah percobaan gagal
    expires_at  TIMESTAMP    NOT NULL,             -- waktu kadaluarsa
    consumed_at TIMESTAMP    NULL,                 -- waktu kode terpakai (NULL = aktif)
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_verification_codes_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_verification_codes_user ON verification_codes (user_id, purpose, created_at);
