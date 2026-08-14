// Package main is the migration CLI. Usage:
//
//	go run ./cmd/migrate up|down|version
//
// It applies the SQL files in ./migrations to the configured database.
package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "time/tzdata" // embed IANA timezone data so APP_TIMEZONE works anywhere
)

const (
	usage           = "usage: migrate <up|down|version>"
	cmdUp           = "up"
	cmdDown         = "down"
	cmdVersion      = "version"
	migrationSource = "file://migrations"
	versionFormat   = "version=%d dirty=%v\n"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		return nil
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	m, err := migrate.New(migrationSource, database.MigrateURL(cfg.Database))
	if err != nil {
		return err
	}
	defer m.Close()

	switch os.Args[1] {
	case cmdUp:
		err = m.Up()
	case cmdDown:
		err = m.Steps(-1)
	case cmdVersion:
		v, dirty, verr := m.Version()
		fmt.Printf(versionFormat, v, dirty)
		err = verr
	default:
		fmt.Println(usage)
		return nil
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	fmt.Println("migrate: ok")
	return nil
}
