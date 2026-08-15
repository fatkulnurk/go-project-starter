package command

import (
	"context"
	"errors"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

func TestUpdatePermission_Execute(t *testing.T) {
	repo := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "posts.edit", Group: "Posts", Name: "Edit Post"})
	uc := NewUpdatePermission(repo, nil, nil)

	if err := uc.Execute(context.Background(), UpdatePermissionCommand{Code: "posts.edit", NewGroup: "Posts", NewName: "Write Post"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updated["p1"] == nil || repo.updated["p1"].Name != "Write Post" {
		t.Fatalf("permission not updated, got %+v", repo.updated)
	}
}

func TestUpdatePermission_GroupChange(t *testing.T) {
	repo := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "posts.edit", Group: "Posts", Name: "Edit Post"})
	uc := NewUpdatePermission(repo, nil, nil)

	if err := uc.Execute(context.Background(), UpdatePermissionCommand{Code: "posts.edit", NewGroup: "Content", NewName: "Edit Post"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updated["p1"].Group != "Content" {
		t.Fatalf("group not updated, got %+v", repo.updated["p1"])
	}
}

func TestUpdatePermission_Protected(t *testing.T) {
	repo := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "rbac.manage", Group: "RBAC", Name: "Manage RBAC"})
	uc := NewUpdatePermission(repo, nil, nil)
	err := uc.Execute(context.Background(), UpdatePermissionCommand{Code: "rbac.manage", NewGroup: "Admin", NewName: "admin.all"})
	if !errors.Is(err, domain.ErrProtected) {
		t.Fatalf("want ErrProtected, got %v", err)
	}
}

func TestUpdatePermission_NotFound(t *testing.T) {
	uc := NewUpdatePermission(newFakePermissionRepo(), nil, nil)
	err := uc.Execute(context.Background(), UpdatePermissionCommand{Code: "ghost", NewGroup: "G", NewName: "x"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestUpdatePermission_Invalid(t *testing.T) {
	uc := NewUpdatePermission(newFakePermissionRepo(), nil, nil)
	err := uc.Execute(context.Background(), UpdatePermissionCommand{Code: "", NewGroup: "G", NewName: "x"})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestUpdatePermission_SameValuesNoop(t *testing.T) {
	repo := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "posts.edit", Group: "Posts", Name: "Edit Post"})
	uc := NewUpdatePermission(repo, nil, nil)
	if err := uc.Execute(context.Background(), UpdatePermissionCommand{Code: "posts.edit", NewGroup: "Posts", NewName: "Edit Post"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.updated) != 0 {
		t.Fatalf("no update expected, got %+v", repo.updated)
	}
}

func TestDeletePermission_Execute(t *testing.T) {
	repo := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "posts.edit", Group: "Posts", Name: "Edit Post"})
	uc := NewDeletePermission(repo, nil, nil)

	if err := uc.Execute(context.Background(), DeletePermissionCommand{Code: "posts.edit"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != "p1" {
		t.Fatalf("permission not deleted, got %+v", repo.deleted)
	}
}

func TestDeletePermission_Protected(t *testing.T) {
	repo := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "rbac.manage", Group: "RBAC", Name: "Manage RBAC"})
	uc := NewDeletePermission(repo, nil, nil)
	err := uc.Execute(context.Background(), DeletePermissionCommand{Code: "rbac.manage"})
	if !errors.Is(err, domain.ErrProtected) {
		t.Fatalf("want ErrProtected, got %v", err)
	}
}

func TestDeletePermission_NotFound(t *testing.T) {
	uc := NewDeletePermission(newFakePermissionRepo(), nil, nil)
	err := uc.Execute(context.Background(), DeletePermissionCommand{Code: "ghost"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestDeletePermission_Invalid(t *testing.T) {
	uc := NewDeletePermission(newFakePermissionRepo(), nil, nil)
	err := uc.Execute(context.Background(), DeletePermissionCommand{Code: " "})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}
