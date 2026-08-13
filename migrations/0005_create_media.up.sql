CREATE TABLE IF NOT EXISTS media (
    id              VARCHAR(64)   NOT NULL PRIMARY KEY,
    model_type      VARCHAR(191)  NOT NULL,
    model_id        VARCHAR(64)   NOT NULL,
    collection_name VARCHAR(64)   NOT NULL DEFAULT 'default',
    name            VARCHAR(255)  NOT NULL,
    file_name       VARCHAR(512)  NOT NULL,
    mime_type       VARCHAR(127)  NOT NULL DEFAULT 'application/octet-stream',
    disk            VARCHAR(32)   NOT NULL,
    size            BIGINT        NOT NULL DEFAULT 0,
    created_at      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_media_model ON media (model_type, model_id, collection_name);
