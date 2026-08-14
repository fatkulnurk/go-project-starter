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
	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib" // postgres driver
)

// New opens a pool for cfg.Database. Callers must Close it.
func New(cfg config.DatabaseConfig) (*sql.DB, error) {
	var db *sql.DB
	var err error
	switch cfg.Driver {
	case config.DriverPostgres:
		db, err = sql.Open("pgx", postgresDSN(cfg))
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", cfg.Driver, err)
		}
	default:
		mc, err := mysqlConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", cfg.Driver, err)
		}
		connector, err := mysql.NewConnector(mc)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", cfg.Driver, err)
		}
		db = sql.OpenDB(connector)
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

// DSN builds a driver-agnostic database/sql DSN for cfg. The postgres value
// is a URL with properly escaped credentials; the mysql value is a
// go-sql-driver DSN. Prefer New, which uses structured configs for both
// drivers so credentials with reserved characters always work.
func DSN(cfg config.DatabaseConfig) (string, string) {
	switch cfg.Driver {
	case config.DriverPostgres:
		return postgresDSN(cfg), "pgx"
	default:
		mc, err := mysqlConfig(cfg)
		if err != nil {
			return "", config.DriverMySQL
		}
		return mc.FormatDSN(), config.DriverMySQL
	}
}

// MigrateURL builds a golang-migrate database URL for cfg. Both drivers parse
// the userinfo with URL rules, so credentials are escaped.
func MigrateURL(cfg config.DatabaseConfig) string {
	switch cfg.Driver {
	case config.DriverPostgres:
		return postgresDSN(cfg)
	default:
		user := url.QueryEscape(cfg.User)
		pass := url.QueryEscape(cfg.Password)
		return fmt.Sprintf(
			"mysql://%s:%s@tcp(%s:%d)/%s?loc=%s&time_zone=%s&tls=%s",
			user, pass, cfg.Host, cfg.Port, url.PathEscape(cfg.Name),
			escapeParam(cfg.TimeZone), mysqlTimeZoneParam(cfg.TimeZone), mysqlTLS(cfg.SSLMode),
		)
	}
}

// postgresDSN returns a postgres:// URL with credentials escaped for the URL
// syntax, which pgx and golang-migrate both decode.
func postgresDSN(cfg config.DatabaseConfig) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   "/" + cfg.Name,
	}
	q := u.Query()
	q.Set("sslmode", pgSSL(cfg.SSLMode))
	q.Set("timezone", cfg.TimeZone)
	u.RawQuery = q.Encode()
	return u.String()
}

// mysqlConfig builds a go-sql-driver Config. User and password are plain
// struct fields, so credentials containing '@', ':', '/' or '?' are safe.
func mysqlConfig(cfg config.DatabaseConfig) (*mysql.Config, error) {
	mc := mysql.NewConfig()
	mc.User = cfg.User
	mc.Passwd = cfg.Password
	mc.Net = "tcp"
	mc.Addr = fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	mc.DBName = cfg.Name
	mc.ParseTime = true
	mc.Params = map[string]string{
		"time_zone": mysqlTimeZoneValue(cfg.TimeZone),
	}
	mc.TLSConfig = mysqlTLS(cfg.SSLMode)
	if cfg.TimeZone != "" && cfg.TimeZone != "UTC" {
		loc, err := time.LoadLocation(cfg.TimeZone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %q: %w", cfg.TimeZone, err)
		}
		mc.Loc = loc
	}
	return mc, nil
}

// escapeParam URL-encodes a query parameter value.
func escapeParam(v string) string { return url.QueryEscape(v) }

// mysqlTimeZoneParam builds the go-sql-driver time_zone parameter, which
// requires the value to be quoted (e.g. time_zone='+00:00'). Named zones
// (e.g. Asia/Jakarta) need the MySQL timezone tables to be loaded, while
// UTC/offset values always work. The returned value is URL-encoded for DSNs.
func mysqlTimeZoneParam(tz string) string {
	return url.QueryEscape(mysqlTimeZoneValue(tz))
}

// mysqlTimeZoneValue returns the raw, quoted session value for time_zone.
func mysqlTimeZoneValue(tz string) string {
	if tz == "UTC" || tz == "" {
		return "'+00:00'"
	}
	return "'" + tz + "'"
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
