package command

import (
	"context"
	"errors"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

func TestSyncRolePermissions_Execute(t *testing.T) {
	roles := newFakeRoleRepo(&domain.Role{ID: "r1", Code: "editor", Name: "Editor"})
	roles.perms["r1"] = []string{"posts.view"}
	perms := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "posts.view", Group: "Posts", Name: "View Post"})
	uc := NewSyncRolePermissions(roles, perms, nil, nil)

	if err := uc.Execute(context.Background(), SyncRolePermissionsCommand{
		Role:        "editor",
		Permissions: []string{"posts.view", "posts.edit"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := perms.byCode["posts.edit"]; !ok {
		t.Fatal("missing permission not created on the fly")
	}
	if perms.byCode["posts.edit"].Group != "posts" {
		t.Fatalf("auto-created permission group = %q, want prefix %q", perms.byCode["posts.edit"].Group, "posts")
	}
}

func TestSyncRolePermissions_NotFound(t *testing.T) {
	uc := NewSyncRolePermissions(newFakeRoleRepo(), newFakePermissionRepo(), nil, nil)
	err := uc.Execute(context.Background(), SyncRolePermissionsCommand{Role: "ghost"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestBootstrap_CreatesMissing(t *testing.T) {
	roles := newFakeRoleRepo()
	perms := newFakePermissionRepo()
	uc := NewBootstrap(roles, perms, nil, nil)

	err := uc.Execute(context.Background(), BootstrapOptions{
		DefaultRoles: []BootstrapRole{
			{Code: "super_admin", Name: "Super Admin"},
			{Code: "user", Name: "User"},
		},
		DefaultPermissions: []BootstrapPermission{
			{Code: "rbac.manage", Group: "RBAC", Name: "Manage RBAC"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if roles.byCode["super_admin"] == nil || roles.byCode["user"] == nil {
		t.Fatalf("roles not seeded: %+v", roles.byCode)
	}
	if roles.byCode["super_admin"].Name != "Super Admin" {
		t.Fatalf("role display name not seeded: %+v", roles.byCode["super_admin"])
	}
	if perms.byCode["rbac.manage"] == nil {
		t.Fatal("permission not seeded")
	}
	if perms.byCode["rbac.manage"].Group != "RBAC" {
		t.Fatalf("permission group not seeded: %+v", perms.byCode["rbac.manage"])
	}
}

func TestBootstrap_Idempotent(t *testing.T) {
	roles := newFakeRoleRepo(&domain.Role{ID: "r1", Code: "user", Name: "User"})
	perms := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "rbac.manage", Group: "RBAC", Name: "Manage RBAC"})
	uc := NewBootstrap(roles, perms, nil, nil)

	err := uc.Execute(context.Background(), BootstrapOptions{
		DefaultRoles: []BootstrapRole{
			{Code: "user", Name: "User"},
		},
		DefaultPermissions: []BootstrapPermission{
			{Code: "rbac.manage", Group: "RBAC", Name: "Manage RBAC"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if roles.byCode["user"].ID != "r1" {
		t.Fatalf("existing role replaced: %+v", roles.byCode["user"])
	}
}
