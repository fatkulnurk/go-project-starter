// Package id defines the shared identifier generator contract. Modules and
// platform packages call New here instead of a UUID library directly, so the
// implementation choice stays centralized and the application layer stays free
// of external libraries.
package id

// Generator produces unique identifiers.
type Generator interface {
	// New returns a version-7 UUID string. UUID v7 is time-ordered so rows sort
	// naturally by insertion time, which keeps indexes friendly.
	New() string
}

// defaultGen is wired by SetDefault in each composition root.
var defaultGen Generator

// New returns a new identifier via the configured generator. It panics when no
// generator has been set, surfacing a wiring mistake instead of minting
// non-UUID ids silently.
func New() string {
	if defaultGen == nil {
		panic("id: SetDefault must be called before New (wire it in the composition root)")
	}
	return defaultGen.New()
}

// SetDefault installs the generator used by New. Call it once during startup.
func SetDefault(g Generator) { defaultGen = g }
