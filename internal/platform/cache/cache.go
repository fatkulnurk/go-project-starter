// Package cache provides cache driver implementations and a factory.
package cache

import (
	"database/sql"
	"fmt"

	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// New returns a cache.Cache for the configured driver. db and dbDriver are
// only used when the driver is "db" and may be nil otherwise.
func New(cfg config.CacheConfig, db *sql.DB, dbDriver string) (cache.Cache, error) {
	switch cfg.Driver {
	case config.DriverRedis:
		return NewRedis(cfg.Redis), nil
	case config.DriverMemory:
		return NewMemory(), nil
	case config.DriverDB:
		if db == nil {
			return nil, fmt.Errorf("cache driver %q requires a database connection", cfg.Driver)
		}
		return NewDatabase(db, dbDriver), nil
	default:
		return nil, fmt.Errorf("unknown cache driver %q", cfg.Driver)
	}
}
