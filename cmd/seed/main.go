// Package main is the seed CLI, modeled after Laravel's artisan db:seed. It
// wires the module seeders (each a small struct with Run, see
// internal/modules/*/seeder) and runs them:
//
//	go run ./cmd/seed            # run every seeder (DatabaseSeeder)
//	go run ./cmd/seed auth.users # run only one seeder (--class)
//
// Seed data is defined as static slices in the module seeder packages (e.g.
// auth/seeder.DefaultUsers), so seeding never depends on fragile
// environment-string parsing. Running the seeders is idempotent and safe to
// re-run.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	_ "time/tzdata" // embed IANA timezone data so APP_TIMEZONE works anywhere

	appid "github.com/fatkulnurk/go-project-starter/internal/application/id"
	"github.com/fatkulnurk/go-project-starter/internal/application/seed"
	authseeder "github.com/fatkulnurk/go-project-starter/internal/modules/auth/seeder"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac"
	rbacseeder "github.com/fatkulnurk/go-project-starter/internal/modules/rbac/seeder"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
	"github.com/fatkulnurk/go-project-starter/internal/platform/hash"
	platformid "github.com/fatkulnurk/go-project-starter/internal/platform/id"
	"github.com/fatkulnurk/go-project-starter/internal/platform/logger"
)

// force reports whether --force was passed, overriding the production guard.
func force() bool {
	for _, a := range os.Args[1:] {
		if a == "--force" {
			return true
		}
	}
	return false
}

func main() {
	if err := run(); err != nil {
		slog.Error("seed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Environment == config.EnvironmentProduction && !force() {
		return errors.New("refusing to seed demo data in production; pass --force to override")
	}
	slog.SetDefault(logger.New(cfg.Environment))
	appid.SetDefault(platformid.Generator{})

	db, err := database.New(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	clk := clock.Real{Loc: cfg.Location()}
	hasher := hash.NewBCrypt(0)

	// --- DatabaseSeeder: register every module seeder --------------------
	reg := seed.New()
	rbacseeder.Register(reg, db, cfg.Database.Driver)
	authseeder.Register(reg, authseeder.Deps{
		DB:       db,
		DBDriver: cfg.Database.Driver,
		Location: cfg.Location(),
		Hasher:   hasher,
		Clock:    clk,
		Roles:    roleAssigner{rbacModule: rbac.New(rbac.Dependencies{DB: db, DBDriver: cfg.Database.Driver})},
	})

	ctx := context.Background()
	args := make([]string, 0, len(os.Args)-1)
	for _, a := range os.Args[1:] {
		if a == "--force" {
			continue
		}
		args = append(args, a)
	}
	if len(args) > 0 {
		return reg.RunOnly(ctx, args...)
	}
	return reg.Run(ctx)
}

// roleAssigner adapts the RBAC service to the auth seeder's narrow Roles port.
// The RBAC module is constructed here with only DB deps; cache and auditor stay
// nil because seeding does not need them.
type roleAssigner struct {
	rbacModule *rbac.Module
}

// AssignRole implements the auth seeder's Roles port by delegating to the
// RBAC service, so demo users can be granted roles during seeding.
func (a roleAssigner) AssignRole(ctx context.Context, userID, roleName string) error {
	return a.rbacModule.Service().AssignRole(ctx, userID, roleName)
}
