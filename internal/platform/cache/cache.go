// Package cache provides cache driver implementations and a factory.
package cache

import (
	"fmt"

	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/redis/go-redis/v9"
)

// New returns a cache.Cache for the configured driver.
func New(cfg config.CacheConfig) (cache.Cache, error) {
	switch cfg.Driver {
	case "redis":
		return NewRedis(cfg.Redis), nil
	case "memory":
		return NewMemory(), nil
	default:
		return nil, fmt.Errorf("unknown cache driver %q", cfg.Driver)
	}
}

var _ = redis.Nil // keep go-redis import pinned for future extensions
