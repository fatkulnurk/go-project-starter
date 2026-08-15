package domain

import (
	"errors"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/application/id"
)

type fakeGenerator struct{ v string }

func (f fakeGenerator) New() string { return f.v }

func init() {
	id.SetDefault(fakeGenerator{v: "fixed-id"})
}

func TestNewRole(t *testing.T) {
	role, err := NewRole("editor", "Editor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role.Code != "editor" {
		t.Fatalf("code = %q, want %q", role.Code, "editor")
	}
	if role.Name != "Editor" {
		t.Fatalf("name = %q, want %q", role.Name, "Editor")
	}
	if role.ID == "" {
		t.Fatal("id must be generated")
	}
}

func TestNewRole_Invalid(t *testing.T) {
	if _, err := NewRole("", "Editor"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
	if _, err := NewRole("editor", " "); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestNewPermission(t *testing.T) {
	p, err := NewPermission("posts.edit", "Posts", "Edit Post")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Code != "posts.edit" {
		t.Fatalf("code = %q, want %q", p.Code, "posts.edit")
	}
	if p.Group != "Posts" {
		t.Fatalf("group = %q, want %q", p.Group, "Posts")
	}
	if p.Name != "Edit Post" {
		t.Fatalf("name = %q, want %q", p.Name, "Edit Post")
	}
	if p.ID == "" {
		t.Fatal("id must be generated")
	}
}

func TestNewPermission_Invalid(t *testing.T) {
	if _, err := NewPermission("", "Posts", "Edit"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
	if _, err := NewPermission("posts.edit", "", " "); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestNewPermission_DefaultGroup(t *testing.T) {
	p, err := NewPermission("posts.edit", "", "Edit Post")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Group != "General" {
		t.Fatalf("group = %q, want %q", p.Group, "General")
	}
}
