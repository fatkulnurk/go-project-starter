// Package database opens a database/sql connection for the configured driver
// (mysql or postgres) and exposes helpers to keep queries portable.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/fatkulnurk/go-project-starter/internal/platform/dbdriver"
	_ "github.com/go-sql-driver/mysql" // mysql driver
	_ "github.com/jackc/pgx/v5/stdlib" // postgres driver
)

// New opens a pool for cfg.Database. Callers must Close it.
func New(cfg config.DatabaseConfig) (*sql.DB, error) {
	dsn, driver := DSN(cfg)
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", cfg.Driver, err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping %s: %w", cfg.Driver, err)
	}
	return db, nil
}

// DSN builds a driver-agnostic database/sql DSN for cfg.
func DSN(cfg config.DatabaseConfig) (string, string) {
	switch cfg.Driver {
	case dbdriver.Postgres:
		return fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
		), "pgx"
	default:
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&loc=UTC",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
		), dbdriver.MySQL
	}
}

// MigrateURL builds a golang-migrate database URL for cfg.
func MigrateURL(cfg config.DatabaseConfig) string {
	switch cfg.Driver {
	case dbdriver.Postgres:
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
	default:
		return fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
	}
}

// Rebind converts '?' placeholders to PostgreSQL '$1' style when driver is
// postgres. For mysql it returns the query unchanged. This keeps repository
// SQL written once with '?' placeholders.
func Rebind(query, driver string) string {
	if driver != dbdriver.Postgres {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	arg := 1
	for _, r := range query {
		if r == '?' {
			b.WriteByte('$')
			b.WriteString(itoa(arg))
			arg++
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
