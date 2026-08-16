// Package seed defines the cross-cutting seeding contract, mirroring
// Laravel's database/seeders: each seeder is a named unit of idempotent data
// initialization exposing Run(ctx), and a registry (like DatabaseSeeder)
// runs them all or a named subset.
package seed

import (
	"context"
	"fmt"
)

// Seed mirrors Laravel's Seeder base class: a named unit of data
// initialization. Run must be idempotent and safe to re-run — existing rows
// are skipped, not failed.
type Seed interface {
	// Run performs the seeding. It must be safe to call repeatedly; returning
	// an error aborts the registry with the seeder's name in the message.
	Run(ctx context.Context) error
}

// Registry collects seeders and runs them, mirroring Laravel's
// DatabaseSeeder::call.
type Registry struct {
	names   []string
	seeders map[string]Seed
}

// New builds an empty registry. Seeders are added with Register before Run,
// and the registry then executes them in registration order.
func New() *Registry {
	return &Registry{seeders: map[string]Seed{}}
}

// Register stores a seeder under a stable name. Duplicate names panic: they
// are a wiring bug that must surface at startup, not at seed time.
func (r *Registry) Register(name string, s Seed) {
	if _, ok := r.seeders[name]; ok {
		panic("seed: duplicate seeder " + name)
	}
	r.seeders[name] = s
	r.names = append(r.names, name)
}

// Run executes every registered seeder in registration order, like running
// php artisan db:seed.
func (r *Registry) Run(ctx context.Context) error {
	return r.RunOnly(ctx, r.names...)
}

// RunOnly executes the named seeders in registration order, like
// php artisan db:seed --class=X. Unknown names fail loudly so typos are
// caught.
func (r *Registry) RunOnly(ctx context.Context, names ...string) error {
	for _, name := range names {
		s, ok := r.seeders[name]
		if !ok {
			return fmt.Errorf("seeder %q: not registered", name)
		}
		if err := s.Run(ctx); err != nil {
			return fmt.Errorf("seeder %s: %w", name, err)
		}
	}
	return nil
}
