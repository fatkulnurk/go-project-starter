// Package seeder holds the auth module's seeders, mirroring Laravel's
// database/seeders: each seeder is a small struct with a Run method that
// seeds one slice of data. Seeders are wired and run by cmd/seed.
package seeder

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"github.com/fatkulnurk/go-project-starter/internal/application/seed"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/infrastructure"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// Roles assigns roles to a seeded user. It is the narrow slice of the RBAC
// service the seeder needs; a nil Roles value disables role assignment.
type Roles interface {
	// AssignRole grants roleName to the user identified by userID, returning an
	// error when the assignment cannot be recorded.
	AssignRole(ctx context.Context, userID, roleName string) error
}

// SeedUser describes a demo account created by UserSeeder. An empty password
// falls back to "password123"; Roles are assigned on top of the default role.
type SeedUser struct {
	Name     string
	Email    string
	Phone    string
	Password string
	// Roles are assigned in addition to the default role.
	Roles []string
}

// DefaultUsers are the demo accounts created by `go run ./cmd/seed`. The admin
// account carries the super_admin role; adjust credentials here, never via
// fragile environment-string parsing.
var DefaultUsers = []SeedUser{
	{Name: "Admin", Email: "admin@example.com", Password: "password123", Roles: []string{authorization.RoleSuperAdmin}},
	{Name: "Demo User", Email: "user@example.com", Password: "password123"},
}

// Register wires the user seeder into a seed registry under "auth.users", so
// `go run ./cmd/seed` can run it by name.
func Register(reg *seed.Registry, deps Deps) {
	reg.Register("auth.users", NewUserSeeder(deps, DefaultUsers))
}

// Deps is the minimal infrastructure UserSeeder needs. It is deliberately
// narrower than the full auth module: seeding demo accounts must not require
// the queue, mailer, SMS, tokens or cache.
type Deps struct {
	DB       *sql.DB
	DBDriver string
	// Location is the app timezone for SQL-written timestamps.
	Location *time.Location
	Hasher   hash.PasswordHasher
	Clock    clock.Clock
	// Roles may be nil; without it only role assignment is skipped.
	Roles Roles
}

// UserSeeder mirrors Laravel's UserSeeder: it creates demo accounts directly
// through the domain, skipping the registration flow so no verification codes
// or deliveries are triggered. Idempotent: existing email/phone is skipped.
type UserSeeder struct {
	deps  Deps
	users []SeedUser
}

// NewUserSeeder builds the seeder around the given deps and demo users. The
// users slice replaces the DefaultUsers set used by Register.
func NewUserSeeder(deps Deps, users []SeedUser) *UserSeeder {
	return &UserSeeder{deps: deps, users: users}
}

// Run implements seed.Seed, seeding every configured user in order. It stops
// at the first failure and reports it.
func (s *UserSeeder) Run(ctx context.Context) error {
	repo := infrastructure.NewUserRepository(s.deps.DB, s.deps.DB, s.deps.DBDriver, s.deps.Location)
	for _, u := range s.users {
		if err := seedOneUser(ctx, s.deps, repo, u); err != nil {
			return err
		}
	}
	return nil
}

var _ seed.Seed = (*UserSeeder)(nil)

func seedOneUser(ctx context.Context, deps Deps, repo domain.UserRepository, u SeedUser) error {
	name := strings.TrimSpace(u.Name)
	email := strings.ToLower(strings.TrimSpace(u.Email))
	phone := strings.TrimSpace(u.Phone)
	password := strings.TrimSpace(u.Password)
	if password == "" {
		password = "password123"
	}

	if email != "" {
		existing, err := repo.FindByEmail(ctx, email)
		if err != nil {
			return err
		}
		if existing != nil {
			slog.Info("user already exists, skipping", "email", email)
			return nil
		}
	}
	if phone != "" {
		existing, err := repo.FindByPhone(ctx, phone)
		if err != nil {
			return err
		}
		if existing != nil {
			slog.Info("user already exists, skipping", "phone", phone)
			return nil
		}
	}

	passwordHash, err := deps.Hasher.Hash(ctx, password)
	if err != nil {
		return err
	}
	user, err := domain.NewUser(name, email, phone, passwordHash, deps.Clock.Now())
	if err != nil {
		return err
	}
	if err := repo.Save(ctx, user); err != nil {
		return err
	}
	slog.Info("user seeded", "user_id", user.ID, "email", email)

	if deps.Roles == nil {
		return nil
	}
	for _, role := range append([]string{authorization.RoleUser}, u.Roles...) {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		if err := deps.Roles.AssignRole(ctx, user.ID, role); err != nil {
			slog.Warn("role assignment skipped", "user_id", user.ID, "role", role, "err", err)
			continue
		}
	}
	return nil
}
