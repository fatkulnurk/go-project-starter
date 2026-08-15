package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/queries"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
	"github.com/go-chi/chi/v5"
)

type stubAuth struct{}

func (stubAuth) Authenticate(ctx context.Context, raw string) (*appauth.Identity, error) {
	return &appauth.Identity{UserID: "u1", Roles: []string{"super_admin"}}, nil
}

type stubAuthz struct{}

func (stubAuthz) HasPermission(context.Context, authorization.Identity, string) error { return nil }
func (stubAuthz) HasRole(context.Context, authorization.Identity, string) error       { return nil }

type fakeRoleRepo struct {
	byCode map[string]*domain.Role
}

func (f *fakeRoleRepo) Save(ctx context.Context, r *domain.Role) error { return nil }
func (f *fakeRoleRepo) FindByCode(ctx context.Context, code string) (*domain.Role, error) {
	return f.byCode[code], nil
}
func (f *fakeRoleRepo) List(ctx context.Context) ([]*domain.Role, error) { return nil, nil }
func (f *fakeRoleRepo) Delete(ctx context.Context, id string) error      { return nil }
func (f *fakeRoleRepo) UpdateName(ctx context.Context, id, name string) error {
	for _, r := range f.byCode {
		if r.ID == id {
			r.Name = name
		}
	}
	return nil
}
func (f *fakeRoleRepo) SetPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	return nil
}
func (f *fakeRoleRepo) PermissionsFor(ctx context.Context, roleID string) ([]string, error) {
	return nil, nil
}

type fakePermissionRepo struct {
	byCode map[string]*domain.Permission
}

func (f *fakePermissionRepo) Save(ctx context.Context, p *domain.Permission) error { return nil }
func (f *fakePermissionRepo) FindByCode(ctx context.Context, code string) (*domain.Permission, error) {
	return f.byCode[code], nil
}
func (f *fakePermissionRepo) List(ctx context.Context) ([]*domain.Permission, error) { return nil, nil }
func (f *fakePermissionRepo) Delete(ctx context.Context, id string) error            { return nil }
func (f *fakePermissionRepo) Update(ctx context.Context, id, group, name string) error {
	for _, p := range f.byCode {
		if p.ID == id {
			p.Group = group
			p.Name = name
		}
	}
	return nil
}

func testRouter(roles *fakeRoleRepo, perms *fakePermissionRepo) http.Handler {
	deps := Deps{
		CreateRole:          commands.NewCreateRole(roles, nil),
		CreatePermission:    commands.NewCreatePermission(perms, nil),
		UpdateRole:          commands.NewUpdateRole(roles, nil, nil),
		DeleteRole:          commands.NewDeleteRole(roles, nil, nil),
		UpdatePermission:    commands.NewUpdatePermission(perms, nil, nil),
		DeletePermission:    commands.NewDeletePermission(perms, nil, nil),
		AssignRole:          commands.NewAssignRole(roles, nil, nil, nil),
		RevokeRole:          commands.NewRevokeRole(roles, nil, nil, nil),
		GrantPermission:     commands.NewGrantPermission(perms, nil, nil, nil),
		RevokePermission:    commands.NewRevokePermission(perms, nil, nil, nil),
		SyncRolePermissions: commands.NewSyncRolePermissions(roles, perms, nil, nil),
		GetRole:             queries.NewGetRole(roles),
		ListRoles:           queries.NewListRoles(roles),
		ListPermissions:     queries.NewListPermissions(perms),
		Authenticator:       stubAuth{},
		Authorizer:          stubAuthz{},
	}
	r := chi.NewRouter()
	RegisterRoutes(r, deps)
	return r
}

func do(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func decodeStatus(rec *httptest.ResponseRecorder) (int, string) {
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &envelope)
	return rec.Code, envelope.Error.Code
}

func TestUpdateRole_Success(t *testing.T) {
	r := testRouter(&fakeRoleRepo{byCode: map[string]*domain.Role{
		"editor": {ID: "r1", Code: "editor", Name: "editor"},
	}}, &fakePermissionRepo{})

	rec := do(r, http.MethodPut, "/api/v1/rbac/roles/editor", `{"name":"writer"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body)
	}
}

func TestUpdateRole_Protected(t *testing.T) {
	r := testRouter(&fakeRoleRepo{byCode: map[string]*domain.Role{
		"super_admin": {ID: "r1", Code: "super_admin", Name: "super_admin"},
	}}, &fakePermissionRepo{})

	rec := do(r, http.MethodPut, "/api/v1/rbac/roles/super_admin", `{"name":"root"}`)
	status, code := decodeStatus(rec)
	if status != http.StatusForbidden || code != "forbidden" {
		t.Fatalf("status/code = %d/%s, want 403/forbidden", status, code)
	}
}

func TestUpdateRole_NotFound(t *testing.T) {
	r := testRouter(&fakeRoleRepo{byCode: map[string]*domain.Role{}}, &fakePermissionRepo{})

	rec := do(r, http.MethodPut, "/api/v1/rbac/roles/ghost", `{"name":"x"}`)
	status, code := decodeStatus(rec)
	if status != http.StatusNotFound || code != "not_found" {
		t.Fatalf("status/code = %d/%s, want 404/not_found", status, code)
	}
}

func TestUpdateRole_SameNameNoop(t *testing.T) {
	r := testRouter(&fakeRoleRepo{byCode: map[string]*domain.Role{
		"editor": {ID: "r1", Code: "editor", Name: "editor"},
	}}, &fakePermissionRepo{})

	rec := do(r, http.MethodPut, "/api/v1/rbac/roles/editor", `{"name":"editor"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body)
	}
}

func TestDeleteRole_Protected(t *testing.T) {
	r := testRouter(&fakeRoleRepo{byCode: map[string]*domain.Role{
		"user": {ID: "r1", Code: "user", Name: "user"},
	}}, &fakePermissionRepo{})

	rec := do(r, http.MethodDelete, "/api/v1/rbac/roles/user", "")
	status, code := decodeStatus(rec)
	if status != http.StatusForbidden || code != "forbidden" {
		t.Fatalf("status/code = %d/%s, want 403/forbidden", status, code)
	}
}

func TestDeleteRole_Success(t *testing.T) {
	r := testRouter(&fakeRoleRepo{byCode: map[string]*domain.Role{
		"editor": {ID: "r1", Code: "editor", Name: "editor"},
	}}, &fakePermissionRepo{})

	rec := do(r, http.MethodDelete, "/api/v1/rbac/roles/editor", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body)
	}
}

func TestUpdatePermission_Success(t *testing.T) {
	r := testRouter(&fakeRoleRepo{}, &fakePermissionRepo{byCode: map[string]*domain.Permission{
		"posts.edit": {ID: "p1", Code: "posts.edit", Group: "Posts", Name: "posts.edit"},
	}})

	rec := do(r, http.MethodPut, "/api/v1/rbac/permissions/posts.edit", `{"group":"Posts","name":"posts.write"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body)
	}
}

func TestDeletePermission_Protected(t *testing.T) {
	r := testRouter(&fakeRoleRepo{}, &fakePermissionRepo{byCode: map[string]*domain.Permission{
		"rbac.manage": {ID: "p1", Code: "rbac.manage", Group: "RBAC", Name: "rbac.manage"},
	}})

	rec := do(r, http.MethodDelete, "/api/v1/rbac/permissions/rbac.manage", "")
	status, code := decodeStatus(rec)
	if status != http.StatusForbidden || code != "forbidden" {
		t.Fatalf("status/code = %d/%s, want 403/forbidden", status, code)
	}
}

func TestGetRole_Success(t *testing.T) {
	r := testRouter(&fakeRoleRepo{byCode: map[string]*domain.Role{
		"editor": {ID: "r1", Code: "editor", Name: "editor"},
	}}, &fakePermissionRepo{})

	rec := do(r, http.MethodGet, "/api/v1/rbac/roles/editor", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body)
	}
	var envelope struct {
		Data roleResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("bad body: %v", err)
	}
	if envelope.Data.ID != "r1" || envelope.Data.Code != "editor" || envelope.Data.Name != "editor" {
		t.Fatalf("unexpected role: %+v", envelope.Data)
	}
}
