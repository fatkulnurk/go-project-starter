package cache

import (
	"context"
	"errors"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/redis/go-redis/v9"
)

// Redis is a go-redis backed cache.
// The client connection settings come from config, and the pool is owned by
// this instance until Close is called.
type Redis struct {
	client *redis.Client
}

// NewRedis builds a Redis cache from config.
// It does not connect eagerly; the first operation establishes the connection.
func NewRedis(cfg config.RedisConfig) *Redis {
	return &Redis{client: redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})}
}

// Ping implements cache.Cache. It pings the server and returns an error when
// the connection is down.
func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Get implements cache.Cache. A missing key returns cache.ErrNotFound
// (redis.Nil is mapped); other errors are returned unwrapped.
func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	v, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, cache.ErrNotFound
	}
	return v, err
}

// Set implements cache.Cache. ttl=0 removes any expiry on the key; a positive
// ttl sets an expiration. Existing keys are overwritten.
func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Delete implements cache.Cache. Removing a missing key is not an error; it
// returns the underlying DEL error, if any.
func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// GetDelete implements cache.Cache via Redis GETDEL, which atomically returns
// the value and removes the key in one server-side operation so a single-use
// token cannot be redeemed twice even under concurrency.
func (r *Redis) GetDelete(ctx context.Context, key string) ([]byte, error) {
	v, err := r.client.GetDel(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, cache.ErrNotFound
	}
	return v, err
}

// Increment implements cache.Cache. It is atomic server-side via INCRBY; a
// missing key starts at zero. The new value and error are returned.
func (r *Redis) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	return r.client.IncrBy(ctx, key, delta).Result()
}

// Expire implements cache.Cache. A non-positive ttl deletes the key; a
// positive ttl sets the key's server-side expiration.
func (r *Redis) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return r.client.Del(ctx, key).Err()
	}
	return r.client.Expire(ctx, key, ttl).Err()
}

// Close implements cache.Cache. It releases the client's pooled connections
// and returns the close error, if any.
func (r *Redis) Close() error { return r.client.Close() }
