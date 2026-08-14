-- =============================================================================
-- TABLE: refresh_tokens
-- Sesi login jangka panjang (refresh token). Access token JWT berumur pendek
-- (15m); saat habis, client menukar refresh token di sini untuk dapat access
-- token baru. Token asli TIDAK disimpan — yang disimpan hanya SHA-256 hash-nya,
-- sehingga bocornya DB tidak membocorkan token. Logout/revoke menandai
-- revoked_at, bukan menghapus baris (agar bisa diaudit).
--
-- Contoh data:
--   id         = '0195c5d5-3a1f-7d00-8000-000000000001'  (UUID v7)
--   user_id    = '0195c5d4-2b40-7d00-8000-000000000001'  (FK -> users.id)
--   token_hash = sha256 hex dari token asli (64 char)
--   expires_at = '2026-02-15 10:30:00'     (TTL default 720h / 30 hari)
--   revoked_at = NULL                       (NULL = masih aktif)
--   created_at = '2026-01-15 10:30:00'
--   updated_at = '2026-01-15 10:30:00'     (ter-update saat revoked)
-- =============================================================================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (dibuat di app)
    user_id    VARCHAR(36)  NOT NULL,             -- pemilik token (FK users.id)
    token_hash VARCHAR(64)  NOT NULL,             -- SHA-256 token asli (unik)
    expires_at TIMESTAMP    NOT NULL,             -- waktu kadaluarsa
    revoked_at TIMESTAMP    NULL,                 -- waktu dicabut/logout (NULL = aktif)
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (token_hash),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id);
