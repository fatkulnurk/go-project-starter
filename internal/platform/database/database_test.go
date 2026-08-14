package database

import (
	"strings"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

func TestMigrateURLEscapesCredentials(t *testing.T) {
	cfg := config.DatabaseConfig{
		Driver:   config.DriverMySQL,
		Host:     "localhost",
		Port:     3306,
		User:     "user@name",
		Password: "p@ss:w/rd",
		Name:     "db/name",
		SSLMode:  "disable",
		TimeZone: "UTC",
	}
	u := MigrateURL(cfg)
	if strings.Contains(u, "user@name") {
		t.Fatalf("mysql user not escaped: %s", u)
	}
	if strings.Contains(u, "p@ss:w/rd") {
		t.Fatalf("mysql password not escaped: %s", u)
	}
	// The raw reserved characters must appear only in encoded form.
	for _, bad := range []string{"user@name", "p@ss:w/rd", "/db/name"} {
		if strings.Contains(u, bad) {
			t.Fatalf("raw credential %q leaked into migrate URL: %s", bad, u)
		}
	}
}

func TestMigrateURLPostgresEscapesPassword(t *testing.T) {
	cfg := config.DatabaseConfig{
		Driver:   config.DriverPostgres,
		Host:     "localhost",
		Port:     5432,
		User:     "app",
		Password: "p@ss:w/rd",
		Name:     "appdb",
		SSLMode:  "disable",
		TimeZone: "UTC",
	}
	u := MigrateURL(cfg)
	if strings.Contains(u, "p@ss:w/rd") {
		t.Fatalf("postgres password not escaped: %s", u)
	}
	if !strings.HasPrefix(u, "postgres://") {
		t.Fatalf("unexpected postgres URL: %s", u)
	}
	if strings.Contains(u, "sslmode=disable") {
		// expected present
	} else {
		t.Fatalf("missing sslmode in postgres URL: %s", u)
	}
}

func TestPostgresDSN(t *testing.T) {
	cfg := config.DatabaseConfig{
		Driver:   config.DriverPostgres,
		Host:     "localhost",
		Port:     5432,
		User:     "app",
		Password: "s3cr3t",
		Name:     "appdb",
		SSLMode:  "disable",
		TimeZone: "Asia/Jakarta",
	}
	u := postgresDSN(cfg)
	if !strings.Contains(u, "timezone=Asia%2FJakarta") {
		t.Fatalf("timezone param not escaped: %s", u)
	}
	if !strings.Contains(u, "postgres://app:s3cr3t@localhost:5432/appdb") {
		t.Fatalf("unexpected DSN: %s", u)
	}
}
