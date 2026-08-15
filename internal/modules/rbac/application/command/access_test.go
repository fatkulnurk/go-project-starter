package command

import (
	"context"
	"errors"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

func TestAssignRole_Execute(t *testing.T) {
	roles := newFakeRoleRepo(&domain.Role{ID: "r1", Code: "editor", Name: "Editor"})
	access := &fakeUserAccessRepo{roles: map[string][]string{}}
	uc := NewAssignRole(roles, access, nil, nil)

	if err := uc.Execute(context.Background(), AssignRoleCommand{UserID: "u1", Role: "editor"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := access.roles["u1"]; len(got) != 1 || got[0] != "r1" {
		t.Fatalf("role not assigned, got %v", got)
	}
}

func TestAssignRole_NotFound(t *testing.T) {
	uc := NewAssignRole(newFakeRoleRepo(), &fakeUserAccessRepo{}, nil, nil)
	err := uc.Execute(context.Background(), AssignRoleCommand{UserID: "u1", Role: "ghost"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestAssignRole_Invalid(t *testing.T) {
	uc := NewAssignRole(newFakeRoleRepo(), &fakeUserAccessRepo{}, nil, nil)
	err := uc.Execute(context.Background(), AssignRoleCommand{UserID: "", Role: ""})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestRevokeRole_Execute(t *testing.T) {
	roles := newFakeRoleRepo(&domain.Role{ID: "r1", Code: "editor", Name: "Editor"})
	access := &fakeUserAccessRepo{roles: map[string][]string{"u1": {"r1"}}}
	uc := NewRevokeRole(roles, access, nil, nil)

	if err := uc.Execute(context.Background(), RevokeRoleCommand{UserID: "u1", Role: "editor"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(access.roles["u1"]) != 0 {
		t.Fatalf("role not revoked, got %v", access.roles["u1"])
	}
}

func TestRevokeRole_NotFound(t *testing.T) {
	uc := NewRevokeRole(newFakeRoleRepo(), &fakeUserAccessRepo{}, nil, nil)
	err := uc.Execute(context.Background(), RevokeRoleCommand{UserID: "u1", Role: "ghost"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestGrantPermission_Execute(t *testing.T) {
	perms := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "posts.edit", Group: "Posts", Name: "Edit Post"})
	access := &fakeUserAccessRepo{directPerms: map[string][]string{}}
	uc := NewGrantPermission(perms, access, nil, nil)

	if err := uc.Execute(context.Background(), GrantPermissionCommand{UserID: "u1", Permission: "posts.edit"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := access.directPerms["u1"]; len(got) != 1 || got[0] != "p1" {
		t.Fatalf("permission not granted, got %v", got)
	}
}

func TestGrantPermission_NotFound(t *testing.T) {
	uc := NewGrantPermission(newFakePermissionRepo(), &fakeUserAccessRepo{}, nil, nil)
	err := uc.Execute(context.Background(), GrantPermissionCommand{UserID: "u1", Permission: "ghost"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestRevokePermission_Execute(t *testing.T) {
	perms := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "posts.edit", Group: "Posts", Name: "Edit Post"})
	access := &fakeUserAccessRepo{directPerms: map[string][]string{"u1": {"p1"}}}
	uc := NewRevokePermission(perms, access, nil, nil)

	if err := uc.Execute(context.Background(), RevokePermissionCommand{UserID: "u1", Permission: "posts.edit"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(access.directPerms["u1"]) != 0 {
		t.Fatalf("permission not revoked, got %v", access.directPerms["u1"])
	}
}

func TestRevokePermission_NotFound(t *testing.T) {
	uc := NewRevokePermission(newFakePermissionRepo(), &fakeUserAccessRepo{}, nil, nil)
	err := uc.Execute(context.Background(), RevokePermissionCommand{UserID: "u1", Permission: "ghost"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
