-- =============================================================================
-- TABLE: media
-- Metadata file yang diunggah user (gambar profil, lampiran, dll). File aslinya
-- disimpan di storage (disk lokal atau S3); tabel ini hanya menyimpan metadata
-- + path. Pola "polimorfik": satu media bisa terkait ke model apa saja lewat
-- model_type + model_id (mis. model_type='user', model_id=id_user).
-- collection_name memisahkan kategori dalam satu model (mis. 'avatar', 'cover').
--
-- Contoh data:
--   id              = '0195c5d8-8f32-7d00-8000-000000000001'  (UUID v7)
--   model_type      = 'user'            (nama kelas/entitas pemilik)
--   model_id        = '0195c5d4-2b40-7d00-8000-000000000001'  (id entitas pemilik)
--   collection_name = 'avatar'          (kategori file milik model tsb)
--   name            = 'profile.jpg'     (nama asli sebelum upload)
--   file_name       = 'profile-20260115-103000.jpg' (nama unik di storage)
--   mime_type       = 'image/jpeg'
--   disk            = 'local' | 's3'
--   size            = 245760            (bytes, ~240 KB)
--   created_at      = '2026-01-15 10:30:00'
--   updated_at      = '2026-01-15 10:30:00'
-- =============================================================================
CREATE TABLE IF NOT EXISTS media (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY, -- UUID v7 (dibuat di app)
    model_type      VARCHAR(191)  NOT NULL,             -- entitas pemilik (mis. 'user')
    model_id        VARCHAR(36)   NOT NULL,             -- id entitas pemilik
    collection_name VARCHAR(64)   NOT NULL DEFAULT 'default', -- kategori (mis. 'avatar')
    name            VARCHAR(255)  NOT NULL,             -- nama file asli
    file_name       VARCHAR(512)  NOT NULL,             -- nama file tersimpan di storage
    mime_type       VARCHAR(127)  NOT NULL DEFAULT 'application/octet-stream', -- tipe MIME
    disk            VARCHAR(32)   NOT NULL,             -- 'local' | 's3' tempat file disimpan
    size            BIGINT        NOT NULL DEFAULT 0,   -- ukuran file dalam bytes
    created_at      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_media_model ON media (model_type, model_id, collection_name);
