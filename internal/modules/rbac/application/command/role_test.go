package command

import (
	"context"
	"errors"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

func TestUpdateRole_Execute(t *testing.T) {
	repo := newFakeRoleRepo(&domain.Role{ID: "r1", Code: "editor", Name: "Editor"})
	uc := NewUpdateRole(repo, nil, nil)

	if err := uc.Execute(context.Background(), UpdateRoleCommand{Code: "editor", NewName: "Writer"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.renamed["r1"] != "Writer" {
		t.Fatalf("role not renamed, got %+v", repo.renamed)
	}
	if repo.byCode["editor"].Name != "Writer" {
		t.Fatalf("role name not updated, got %+v", repo.byCode["editor"])
	}
}

func TestUpdateRole_Protected(t *testing.T) {
	for _, code := range []string{"super_admin", "user"} {
		repo := newFakeRoleRepo(&domain.Role{ID: "r1", Code: code, Name: code})
		uc := NewUpdateRole(repo, nil, nil)
		err := uc.Execute(context.Background(), UpdateRoleCommand{Code: code, NewName: "renamed"})
		if !errors.Is(err, domain.ErrProtected) {
			t.Fatalf("%q: want ErrProtected, got %v", code, err)
		}
	}
}

func TestUpdateRole_NotFound(t *testing.T) {
	uc := NewUpdateRole(newFakeRoleRepo(), nil, nil)
	err := uc.Execute(context.Background(), UpdateRoleCommand{Code: "ghost", NewName: "x"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestUpdateRole_Invalid(t *testing.T) {
	uc := NewUpdateRole(newFakeRoleRepo(), nil, nil)
	err := uc.Execute(context.Background(), UpdateRoleCommand{Code: " ", NewName: ""})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestUpdateRole_SameNameNoop(t *testing.T) {
	repo := newFakeRoleRepo(&domain.Role{ID: "r1", Code: "editor", Name: "Editor"})
	uc := NewUpdateRole(repo, nil, nil)
	if err := uc.Execute(context.Background(), UpdateRoleCommand{Code: "editor", NewName: "Editor"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.renamed) != 0 {
		t.Fatalf("no update expected, got %+v", repo.renamed)
	}
}

func TestDeleteRole_Execute(t *testing.T) {
	repo := newFakeRoleRepo(&domain.Role{ID: "r1", Code: "editor", Name: "Editor"})
	uc := NewDeleteRole(repo, nil, nil)

	if err := uc.Execute(context.Background(), DeleteRoleCommand{Code: "editor"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != "r1" {
		t.Fatalf("role not deleted, got %+v", repo.deleted)
	}
}

func TestDeleteRole_Protected(t *testing.T) {
	for _, code := range []string{"super_admin", "user"} {
		repo := newFakeRoleRepo(&domain.Role{ID: "r1", Code: code, Name: code})
		uc := NewDeleteRole(repo, nil, nil)
		err := uc.Execute(context.Background(), DeleteRoleCommand{Code: code})
		if !errors.Is(err, domain.ErrProtected) {
			t.Fatalf("%q: want ErrProtected, got %v", code, err)
		}
	}
}

func TestDeleteRole_NotFound(t *testing.T) {
	uc := NewDeleteRole(newFakeRoleRepo(), nil, nil)
	err := uc.Execute(context.Background(), DeleteRoleCommand{Code: "ghost"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestDeleteRole_Invalid(t *testing.T) {
	uc := NewDeleteRole(newFakeRoleRepo(), nil, nil)
	err := uc.Execute(context.Background(), DeleteRoleCommand{Code: "  "})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}
