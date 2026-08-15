package queries

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	rbaccache "github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

func TestGetRole_Execute(t *testing.T) {
	repo := newFakeRoleRepo(&domain.Role{ID: "r1", Code: "editor", Name: "Editor"})
	repo.perms["r1"] = []string{"posts.edit", "posts.view"}
	q := NewGetRole(repo)

	res, err := q.Execute(context.Background(), "editor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "r1" || res.Code != "editor" || res.Name != "Editor" {
		t.Fatalf("unexpected role: %+v", res)
	}
	if want := []string{"posts.edit", "posts.view"}; !reflect.DeepEqual(res.Permissions, want) {
		t.Fatalf("permissions = %v, want %v", res.Permissions, want)
	}
}

func TestGetRole_NotFound(t *testing.T) {
	q := NewGetRole(newFakeRoleRepo())
	_, err := q.Execute(context.Background(), "ghost")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestListRoles_Execute(t *testing.T) {
	repo := newFakeRoleRepo(
		&domain.Role{ID: "r1", Code: "editor", Name: "Editor"},
		&domain.Role{ID: "r2", Code: "viewer", Name: "Viewer"},
	)
	repo.perms["r1"] = []string{"posts.edit"}
	q := NewListRoles(repo)

	res, err := q.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("got %d roles, want 2", len(res))
	}
	for _, r := range res {
		switch r.Code {
		case "editor":
			if want := []string{"posts.edit"}; !reflect.DeepEqual(r.Permissions, want) {
				t.Fatalf("editor permissions = %v, want %v", r.Permissions, want)
			}
		case "viewer":
			if len(r.Permissions) != 0 {
				t.Fatalf("viewer permissions = %v, want none", r.Permissions)
			}
		default:
			t.Fatalf("unexpected role %q", r.Code)
		}
	}
}

func TestListPermissions_Execute(t *testing.T) {
	repo := &fakePermissionRepo{perms: []*domain.Permission{
		{ID: "p1", Code: "posts.edit", Group: "Posts", Name: "Edit Post"},
		{ID: "p2", Code: "posts.view", Group: "Posts", Name: "View Post"},
	}}
	q := NewListPermissions(repo)

	res, err := q.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("got %d permissions, want 2", len(res))
	}
}

func TestGetUser_MergesDirectAndInherited(t *testing.T) {
	access := &fakeUserAccessRepo{
		roles:       []string{"editor"},
		directPerms: []string{"posts.edit"},
		inherited:   []string{"posts.view", "posts.edit"},
	}
	q := NewGetUser(access, nil)

	res, err := q.Execute(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"editor"}; !reflect.DeepEqual(res.Roles, want) {
		t.Fatalf("roles = %v, want %v", res.Roles, want)
	}
	want := []string{"posts.edit", "posts.view"}
	if !reflect.DeepEqual(res.Permissions, want) {
		t.Fatalf("permissions = %v, want %v", res.Permissions, want)
	}
}

func TestGetUser_PropagatesError(t *testing.T) {
	access := &fakeUserAccessRepo{loadRolesErr: errors.New("db down")}
	q := NewGetUser(access, nil)
	if _, err := q.Execute(context.Background(), "u1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetUser_WithCache(t *testing.T) {
	access := &fakeUserAccessRepo{
		roles:       []string{"editor"},
		directPerms: []string{"posts.edit"},
		inherited:   []string{"posts.view"},
	}
	q := NewGetUser(access, rbaccache.NewPermissionCache(newFakeCache(), time.Minute))

	res1, err := q.Execute(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Mutate the source: a cache hit must still return the cached snapshot.
	access.roles = []string{"changed"}
	access.directPerms = []string{"changed"}
	access.inherited = []string{"changed"}

	res2, err := q.Execute(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(res2, res1) {
		t.Fatalf("cache miss returned different data: %+v vs %+v", res2, res1)
	}
}
