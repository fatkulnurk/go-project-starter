// Package database opens a database/sql connection for the configured driver
// (mysql or postgres) and exposes helpers to keep queries portable.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
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
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

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
	case config.DriverPostgres:
		return fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s&timezone=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, pgSSL(cfg.SSLMode), escapeParam(cfg.TimeZone),
		), "pgx"
	default:
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&loc=%s&time_zone=%s&tls=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
			escapeParam(cfg.TimeZone), mysqlTimeZoneParam(cfg.TimeZone), mysqlTLS(cfg.SSLMode),
		), config.DriverMySQL
	}
}

// MigrateURL builds a golang-migrate database URL for cfg.
func MigrateURL(cfg config.DatabaseConfig) string {
	switch cfg.Driver {
	case config.DriverPostgres:
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&timezone=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, pgSSL(cfg.SSLMode), escapeParam(cfg.TimeZone))
	default:
		return fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?loc=%s&time_zone=%s&tls=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
			escapeParam(cfg.TimeZone), mysqlTimeZoneParam(cfg.TimeZone), mysqlTLS(cfg.SSLMode))
	}
}

// escapeParam URL-encodes a query parameter value.
func escapeParam(v string) string { return url.QueryEscape(v) }

// mysqlTimeZoneParam builds the go-sql-driver time_zone parameter, which
// requires the value to be quoted (e.g. time_zone='+00:00'). Named zones
// (e.g. Asia/Jakarta) need the MySQL timezone tables to be loaded, while
// UTC/offset values always work.
func mysqlTimeZoneParam(tz string) string {
	if tz == "UTC" || tz == "" {
		return "%27%2B00%3A00%27" // '+00:00'
	}
	return "%27" + url.QueryEscape(tz) + "%27"
}

// pgSSL maps DB_SSL_MODE to a postgres sslmode value.
func pgSSL(mode string) string {
	switch mode {
	case "", "disable":
		return "disable"
	case "require":
		return "require"
	case "verify-ca":
		return "verify-ca"
	case "verify-full":
		return "verify-full"
	default:
		return "disable"
	}
}

// mysqlTLS maps DB_SSL_MODE to a go-sql-driver tls parameter.
func mysqlTLS(mode string) string {
	switch mode {
	case "require":
		return "true"
	case "skip-verify":
		return "skip-verify"
	default:
		return "false"
	}
}

// Rebind converts '?' placeholders to PostgreSQL '$1' style when driver is
// postgres. For mysql it returns the query unchanged. This keeps repository
// SQL written once with '?' placeholders.
func Rebind(query, driver string) string {
	if driver != config.DriverPostgres {
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
