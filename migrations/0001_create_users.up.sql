-- =============================================================================
-- TABLE: users
-- Akun pengguna aplikasi. Satu baris = satu user (email dan/atau nomor HP).
-- Status menandakan akun aktif atau diblokir; kolom *verified_at menandakan
-- email/HP sudah diverifikasi OTP. Semua kolom id berisi UUID v7 (36 char)
-- yang dibuat di aplikasi, bukan di database.
--
-- Contoh data:
--   id                = '0195c5d4-2b40-7d00-8000-000000000001'  (UUID v7)
--   name              = 'Budi Santoso'
--   email             = 'budi@example.com'
--   phone             = '+6281234567890'       (NULL jika tidak ada)
--   password_hash     = '$2a$10$...'           (bcrypt/argon2 hash)
--   email_verified_at = '2026-01-15 10:30:00'  (NULL = belum verifikasi)
--   phone_verified_at = NULL
--   status            = 'active' | 'suspended'
--   created_at        = '2026-01-15 10:30:00'
--   updated_at        = '2026-01-15 10:30:00'
-- =============================================================================
CREATE TABLE IF NOT EXISTS users (
    id                VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (dibuat di app)
    name              VARCHAR(255) NOT NULL,             -- nama tampilan user
    email             VARCHAR(255) NULL,                 -- email login (unik, boleh kosong jika pakai HP)
    phone             VARCHAR(32)  NULL,                 -- nomor HP login (unik, boleh kosong jika pakai email)
    password_hash     VARCHAR(255) NOT NULL,             -- hash password, tidak pernah plaintext
    email_verified_at TIMESTAMP    NULL,                 -- waktu email diverifikasi (NULL = belum)
    phone_verified_at TIMESTAMP    NULL,                 -- waktu HP diverifikasi (NULL = belum)
    status            VARCHAR(20)  NOT NULL DEFAULT 'active', -- 'active' | 'suspended'
    created_at        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (email),
    UNIQUE (phone)
);
