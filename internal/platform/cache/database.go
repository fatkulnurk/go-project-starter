package cache

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
)

// Database is a cache backed by the application SQL database. The schema
// follows Laravel: cache_key + value + expiration (unix seconds; 0 = never
// expires). Values are stored as TEXT, so they must be textual (counters,
// JSON, ...). Useful when Redis is not available; keep in mind every read
// round-trips the database, so prefer redis in production.
type Database struct {
	db     *sql.DB
	driver string
}

// NewDatabase builds a database-backed cache on the shared pool. It does not
// take ownership of db: Close is a no-op.
func NewDatabase(db *sql.DB, driver string) *Database {
	return &Database{db: db, driver: driver}
}

// Ping implements cache.Cache.
func (d *Database) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// Get implements cache.Cache.
func (d *Database) Get(ctx context.Context, key string) ([]byte, error) {
	const q = `SELECT value FROM cache WHERE cache_key = ? AND (expiration = 0 OR expiration > ?)`
	var v string
	err := d.db.QueryRowContext(ctx, d.q(q), key, time.Now().UTC().Unix()).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, cache.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return []byte(v), nil
}

// Set implements cache.Cache. ttl=0 stores the value without an expiry.
func (d *Database) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	expiration := int64(0)
	if ttl > 0 {
		expiration = time.Now().UTC().Unix() + int64(ttl.Seconds())
	}
	var q string
	if d.driver == config.DriverPostgres {
		q = `INSERT INTO cache (cache_key, value, expiration, created_at, updated_at)
			 VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			 ON CONFLICT (cache_key) DO UPDATE SET value = EXCLUDED.value,
				expiration = EXCLUDED.expiration, updated_at = CURRENT_TIMESTAMP`
	} else {
		q = `INSERT INTO cache (cache_key, value, expiration, created_at, updated_at)
			 VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			 ON DUPLICATE KEY UPDATE value = VALUES(value),
				expiration = VALUES(expiration), updated_at = CURRENT_TIMESTAMP`
	}
	_, err := d.db.ExecContext(ctx, d.q(q), key, string(value), expiration)
	return err
}

// Delete implements cache.Cache.
func (d *Database) Delete(ctx context.Context, key string) error {
	const q = `DELETE FROM cache WHERE cache_key = ?`
	_, err := d.db.ExecContext(ctx, d.q(q), key)
	return err
}

// Increment implements cache.Cache. It creates the key with delta when missing.
// A transaction with a row lock keeps concurrent increments race-free: the row
// is seeded idempotently first so two first-touches cannot double-insert.
func (d *Database) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck // rolled back on error only

	var seedQ string
	if d.driver == config.DriverPostgres {
		seedQ = `INSERT INTO cache (cache_key, value, expiration, created_at, updated_at)
			 VALUES (?, '0', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			 ON CONFLICT (cache_key) DO NOTHING`
	} else {
		seedQ = `INSERT IGNORE INTO cache (cache_key, value, expiration, created_at, updated_at)
			 VALUES (?, '0', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`
	}
	if _, err := tx.ExecContext(ctx, d.q(seedQ), key); err != nil {
		return 0, err
	}

	var current sql.NullString
	const selectQ = `SELECT value FROM cache WHERE cache_key = ? FOR UPDATE`
	err = tx.QueryRowContext(ctx, d.q(selectQ), key).Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	var n int64
	if current.Valid {
		n, err = strconv.ParseInt(current.String, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	n += delta

	const updateQ = `UPDATE cache SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE cache_key = ?`
	if _, err := tx.ExecContext(ctx, d.q(updateQ), strconv.FormatInt(n, 10), key); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return n, nil
}

// Expire implements cache.Cache. A zero ttl removes the expiry.
func (d *Database) Expire(ctx context.Context, key string, ttl time.Duration) error {
	expiration := int64(0)
	if ttl > 0 {
		expiration = time.Now().UTC().Unix() + int64(ttl.Seconds())
	}
	const q = `UPDATE cache SET expiration = ?, updated_at = CURRENT_TIMESTAMP WHERE cache_key = ?`
	_, err := d.db.ExecContext(ctx, d.q(q), expiration, key)
	return err
}

// Close implements cache.Cache. The SQL pool is owned by the composition root,
// so nothing is released here.
func (d *Database) Close() error { return nil }

func (d *Database) q(query string) string { return database.Rebind(query, d.driver) }
