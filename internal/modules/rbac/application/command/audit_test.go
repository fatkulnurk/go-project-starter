package command

import (
	"context"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

func TestAuditAndBump(t *testing.T) {
	ctx := context.Background()
	roles := newFakeRoleRepo(&domain.Role{ID: "r1", Code: "editor", Name: "Editor"})
	perms := newFakePermissionRepo(&domain.Permission{ID: "p1", Code: "posts.view", Group: "Posts", Name: "View Post"})
	access := &fakeUserAccessRepo{roles: map[string][]string{}, directPerms: map[string][]string{}}
	aud := &fakeAuditor{}
	bumper := &fakeBumper{}

	cases := []struct {
		name   string
		run    func() error
		action audit.Action
	}{
		{"create role", func() error {
			return NewCreateRole(newFakeRoleRepo(), aud).Execute(ctx, CreateRoleCommand{Code: "viewer", Name: "Viewer"})
		}, audit.ActionCreated},
		{"create permission", func() error {
			return NewCreatePermission(newFakePermissionRepo(), aud).Execute(ctx, CreatePermissionCommand{Code: "posts.delete", Group: "Posts", Name: "Delete Post"})
		}, audit.ActionCreated},
		{"assign role", func() error {
			return NewAssignRole(roles, access, bumper, aud).Execute(ctx, AssignRoleCommand{UserID: "u1", Role: "editor"})
		}, audit.ActionCreated},
		{"revoke role", func() error {
			return NewRevokeRole(roles, access, bumper, aud).Execute(ctx, RevokeRoleCommand{UserID: "u1", Role: "editor"})
		}, audit.ActionDeleted},
		{"grant permission", func() error {
			return NewGrantPermission(perms, access, bumper, aud).Execute(ctx, GrantPermissionCommand{UserID: "u1", Permission: "posts.view"})
		}, audit.ActionCreated},
		{"revoke permission", func() error {
			return NewRevokePermission(perms, access, bumper, aud).Execute(ctx, RevokePermissionCommand{UserID: "u1", Permission: "posts.view"})
		}, audit.ActionDeleted},
		{"sync role permissions", func() error {
			return NewSyncRolePermissions(roles, perms, bumper, aud).Execute(ctx, SyncRolePermissionsCommand{Role: "editor", Permissions: []string{"posts.view"}})
		}, audit.ActionUpdated},
		{"update role", func() error {
			return NewUpdateRole(roles, bumper, aud).Execute(ctx, UpdateRoleCommand{Code: "editor", NewName: "Writer"})
		}, audit.ActionUpdated},
		{"delete role", func() error {
			return NewDeleteRole(roles, bumper, aud).Execute(ctx, DeleteRoleCommand{Code: "editor"})
		}, audit.ActionDeleted},
		{"update permission", func() error {
			return NewUpdatePermission(perms, bumper, aud).Execute(ctx, UpdatePermissionCommand{Code: "posts.view", NewGroup: "Posts", NewName: "Read Post"})
		}, audit.ActionUpdated},
		{"delete permission", func() error {
			return NewDeletePermission(perms, bumper, aud).Execute(ctx, DeletePermissionCommand{Code: "posts.view"})
		}, audit.ActionDeleted},
		{"bootstrap", func() error {
			if err := roles.Save(ctx, &domain.Role{ID: "r-user", Code: "user", Name: "User"}); err != nil {
				return err
			}
			return NewBootstrap(roles, perms, bumper, aud).Execute(ctx, BootstrapOptions{
				DefaultRoles:       []BootstrapRole{{Code: "user", Name: "User"}},
				DefaultPermissions: []BootstrapPermission{{Code: "rbac.manage", Group: "RBAC", Name: "Manage RBAC"}},
			})
		}, audit.ActionCreated},
	}

	for _, tc := range cases {
		before := len(aud.entries)
		if err := tc.run(); err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		after := len(aud.entries)
		if after != before+1 {
			t.Fatalf("%s: expected 1 audit entry, got %d", tc.name, after-before)
		}
		if got := aud.entries[before].Action; got != tc.action {
			t.Fatalf("%s: action = %q, want %q", tc.name, got, tc.action)
		}
	}
	if bumper.bumped == 0 {
		t.Fatal("expected cache bump on at least one mutation")
	}
}
