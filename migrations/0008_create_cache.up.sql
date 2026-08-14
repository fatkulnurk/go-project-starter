-- =============================================================================
-- TABLE: cache
-- Backing store untuk cache driver berbasis database (CACHE_DRIVER=db). Skema
-- mengikuti Laravel: kolom expiration menyimpan unix timestamp (detik) kapan
-- entri kedaluwarsa, nilai 0 berarti tidak pernah kadaluwarsa. value bertipe
-- TEXT agar portabel antara MySQL dan PostgreSQL (mediumText di Laravel).
--
-- Contoh data:
--   cache_key  = 'auth:ratelimit:192.168.1.10'
--   value      = '3'                          (counter rate-limit / JSON)
--   expiration = 1768449600                   (unix detik; 0 = abadi)
--   created_at = '2026-01-15 10:30:00'
--   updated_at = '2026-01-15 10:30:00'        (ter-update tiap Set/Increment)
-- =============================================================================
CREATE TABLE IF NOT EXISTS cache (
    cache_key  VARCHAR(255) NOT NULL PRIMARY KEY, -- kunci cache
    value      TEXT         NOT NULL,             -- nilai cache (string/JSON)
    expiration BIGINT       NOT NULL DEFAULT 0,   -- unix detik kadaluarsa; 0 = abadi
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_cache_expiration ON cache (expiration);