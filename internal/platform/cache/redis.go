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
type Redis struct {
	client *redis.Client
}

// NewRedis builds a Redis cache from config.
func NewRedis(cfg config.RedisConfig) *Redis {
	return &Redis{client: redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})}
}

// Get implements cache.Cache.
func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	v, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, cache.ErrNotFound
	}
	return v, err
}

// Set implements cache.Cache.
func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Delete implements cache.Cache.
func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Increment implements cache.Cache.
func (r *Redis) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	return r.client.IncrBy(ctx, key, delta).Result()
}

// Expire implements cache.Cache.
func (r *Redis) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

// Close implements cache.Cache.
func (r *Redis) Close() error { return r.client.Close() }
