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

// Ping implements cache.Cache. It reports whether the underlying SQL pool is
// reachable and returns the database/sql error otherwise.
func (d *Database) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// Get implements cache.Cache. It returns cache.ErrNotFound when the key is
// missing or its expiration has passed; otherwise it returns the stored value.
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

// Set implements cache.Cache. ttl=0 stores the value without an expiry; a
// positive ttl is rounded up to whole seconds. Existing keys are overwritten.
func (d *Database) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	expiration := int64(0)
	if ttl > 0 {
		expiration = time.Now().UTC().Unix() + ceilSeconds(ttl)
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

// Delete implements cache.Cache. Removing a missing key is not an error; it
// returns the DELETE statement's error, if any.
func (d *Database) Delete(ctx context.Context, key string) error {
	const q = `DELETE FROM cache WHERE cache_key = ?`
	_, err := d.db.ExecContext(ctx, d.q(q), key)
	return err
}

// GetDelete implements cache.Cache. The read and delete happen inside one
// transaction with a row lock, so a concurrent GetDelete cannot double-redeem
// a single-use token. Expired rows are treated as missing.
func (d *Database) GetDelete(ctx context.Context, key string) ([]byte, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // rolled back on error only

	var v string
	const selectQ = `SELECT value FROM cache WHERE cache_key = ? AND (expiration = 0 OR expiration > ?) FOR UPDATE`
	err = tx.QueryRowContext(ctx, d.q(selectQ), key, time.Now().UTC().Unix()).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, cache.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	const delQ = `DELETE FROM cache WHERE cache_key = ?`
	if _, err := tx.ExecContext(ctx, d.q(delQ), key); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return []byte(v), nil
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
	var expiration int64
	// The locking read only sees live rows (expiration 0 = never, or still in
	// the future). A row whose expiration passed is treated as missing so the
	// counter resets and the rate limiter does not lock out permanently.
	const selectQ = `SELECT value, expiration FROM cache WHERE cache_key = ? AND (expiration = 0 OR expiration > ?) FOR UPDATE`
	err = tx.QueryRowContext(ctx, d.q(selectQ), key, time.Now().UTC().Unix()).Scan(&current, &expiration)
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

	// A live row keeps its expiration; an expired or missing row is reset to a
	// fresh counter without an expiry (the caller sets one via Expire).
	if !current.Valid {
		expiration = 0
	}

	const updateQ = `UPDATE cache SET value = ?, expiration = ?, updated_at = CURRENT_TIMESTAMP WHERE cache_key = ?`
	if _, err := tx.ExecContext(ctx, d.q(updateQ), strconv.FormatInt(n, 10), expiration, key); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return n, nil
}

// Expire implements cache.Cache. A non-positive ttl removes the key (via
// Delete); otherwise it extends the key's expiration in whole seconds.
func (d *Database) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return d.Delete(ctx, key)
	}
	expiration := time.Now().UTC().Unix() + ceilSeconds(ttl)
	const q = `UPDATE cache SET expiration = ?, updated_at = CURRENT_TIMESTAMP WHERE cache_key = ?`
	_, err := d.db.ExecContext(ctx, d.q(q), expiration, key)
	return err
}

// ceilSeconds rounds a duration up to whole seconds so sub-second TTLs never
// truncate to zero (which would mean "never expires").
func ceilSeconds(ttl time.Duration) int64 {
	return (int64(ttl) + int64(time.Second) - 1) / int64(time.Second)
}

// Close implements cache.Cache. The SQL pool is owned by the composition root,
// so nothing is released here.
func (d *Database) Close() error { return nil }

func (d *Database) q(query string) string { return database.Rebind(query, d.driver) }
