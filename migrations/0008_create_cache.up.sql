-- =============================================================================
-- TABLE: cache
-- Backing store for the database-based cache driver (CACHE_DRIVER=db). The
-- schema follows Laravel: expiration stores the unix timestamp (seconds) when
-- the entry expires, 0 means never expires. value is TEXT so it stays portable
-- between MySQL and PostgreSQL (mediumText in Laravel).
--
-- Example data:
--   cache_key  = 'auth:ratelimit:192.168.1.10'
--   value      = '3'                          (rate-limit counter / JSON)
--   expiration = 1768449600                   (unix seconds; 0 = permanent)
--   created_at = '2026-01-15 10:30:00'
--   updated_at = '2026-01-15 10:30:00'        (updated on every Set/Increment)
-- =============================================================================
CREATE TABLE IF NOT EXISTS cache (
    cache_key  VARCHAR(255) NOT NULL PRIMARY KEY, -- cache key
    value      TEXT         NOT NULL,             -- cache value (string/JSON)
    expiration BIGINT       NOT NULL DEFAULT 0,   -- unix seconds until expiry; 0 = permanent
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_cache_expiration ON cache (expiration);