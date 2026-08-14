-- =============================================================================
-- TABLE: media
-- Metadata for files uploaded by users (profile pictures, attachments, etc.).
-- The actual file lives in storage (local disk or S3); this table only holds
-- metadata + path. "Polymorphic" pattern: one media record can relate to any
-- model via model_type + model_id (e.g. model_type='user', model_id=user_id).
-- collection_name separates categories within one model (e.g. 'avatar', 'cover').
--
-- Example data:
--   id              = '0195c5d8-8f32-7d00-8000-000000000001'  (UUID v7)
--   model_type      = 'user'            (owner entity class name)
--   model_id        = '0195c5d4-2b40-7d00-8000-000000000001'  (owner entity id)
--   collection_name = 'avatar'          (file category for that model)
--   name            = 'profile.jpg'     (original name before upload)
--   file_name       = 'profile-20260115-103000.jpg' (unique name in storage)
--   mime_type       = 'image/jpeg'
--   disk            = 'local' | 's3'
--   size            = 245760            (bytes, ~240 KB)
--   created_at      = '2026-01-15 10:30:00'
--   updated_at      = '2026-01-15 10:30:00'
-- =============================================================================
CREATE TABLE IF NOT EXISTS media (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY, -- UUID v7 (generated in app)
    model_type      VARCHAR(191)  NOT NULL,             -- owner entity (e.g. 'user')
    model_id        VARCHAR(36)   NOT NULL,             -- owner entity id
    collection_name VARCHAR(64)   NOT NULL DEFAULT 'default', -- category (e.g. 'avatar')
    name            VARCHAR(255)  NOT NULL,             -- original file name
    file_name       VARCHAR(512)  NOT NULL,             -- stored file name in storage
    mime_type       VARCHAR(127)  NOT NULL DEFAULT 'application/octet-stream', -- MIME type
    disk            VARCHAR(32)   NOT NULL,             -- 'local' | 's3' where the file lives
    size            BIGINT        NOT NULL DEFAULT 0,   -- file size in bytes
    created_at      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_media_model ON media (model_type, model_id, collection_name);
