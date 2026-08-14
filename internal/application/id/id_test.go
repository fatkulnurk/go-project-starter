package id

import (
	"testing"
)

// fakeGenerator returns a fixed value so tests never depend on uuid.
type fakeGenerator struct{ v string }

func (f fakeGenerator) New() string { return f.v }

func TestNewPanicsWithoutDefault(t *testing.T) {
	prev := defaultGen
	SetDefault(nil)
	t.Cleanup(func() { SetDefault(prev) })

	defer func() {
		if recover() == nil {
			t.Fatal("New() did not panic without a configured generator")
		}
	}()
	_ = New()
}

func TestNewUsesSetDefault(t *testing.T) {
	prev := defaultGen
	SetDefault(fakeGenerator{v: "fixed-id"})
	t.Cleanup(func() { SetDefault(prev) })

	if got := New(); got != "fixed-id" {
		t.Fatalf("New() = %q, want %q", got, "fixed-id")
	}
}
