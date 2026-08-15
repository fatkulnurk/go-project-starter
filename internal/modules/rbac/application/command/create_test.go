package command

import (
	"context"
	"errors"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

func TestCreateRole_Execute(t *testing.T) {
	repo := newFakeRoleRepo()
	uc := NewCreateRole(repo, nil)

	if err := uc.Execute(context.Background(), CreateRoleCommand{Code: "editor", Name: "Editor"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.byCode["editor"] == nil {
		t.Fatal("role not saved")
	}
}

func TestCreateRole_Conflict(t *testing.T) {
	repo := newFakeRoleRepo(&domain.Role{ID: "r1", Code: "editor", Name: "Editor"})
	uc := NewCreateRole(repo, nil)
	err := uc.Execute(context.Background(), CreateRoleCommand{Code: "editor", Name: "Editor"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestCreateRole_Invalid(t *testing.T) {
	uc := NewCreateRole(newFakeRoleRepo(), nil)
	if err := uc.Execute(context.Background(), CreateRoleCommand{Code: " ", Name: "Editor"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
	if err := uc.Execute(context.Background(), CreateRoleCommand{Code: "editor", Name: ""}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestCreatePermission_Execute(t *testing.T) {
	repo := newFakePermissionRepo()
	uc := NewCreatePermission(repo, nil)

	if err := uc.Execute(context.Background(), CreatePermissionCommand{Code: "posts.edit", Group: "Posts", Name: "Edit Post"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.byCode["posts.edit"] == nil {
		t.Fatal("permission not saved")
	}
}

func TestCreatePermission_Conflict(t *testing.T) {
	repo := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "posts.edit", Group: "Posts", Name: "Edit Post"})
	uc := NewCreatePermission(repo, nil)
	err := uc.Execute(context.Background(), CreatePermissionCommand{Code: "posts.edit", Group: "Posts", Name: "Edit Post"})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestCreatePermission_Invalid(t *testing.T) {
	uc := NewCreatePermission(newFakePermissionRepo(), nil)
	err := uc.Execute(context.Background(), CreatePermissionCommand{Code: "", Group: "Posts", Name: "Edit Post"})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}
