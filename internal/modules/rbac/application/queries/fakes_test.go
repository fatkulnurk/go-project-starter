package queries

import (
	"context"
	"encoding/json"
	"time"

	appcache "github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

type fakeCache struct {
	data map[string][]byte
}

func newFakeCache() *fakeCache { return &fakeCache{data: map[string][]byte{}} }

func (f *fakeCache) Get(ctx context.Context, key string) ([]byte, error) {
	if v, ok := f.data[key]; ok {
		return v, nil
	}
	return nil, appcache.ErrNotFound
}

func (f *fakeCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	f.data[key] = value
	return nil
}

func (f *fakeCache) Delete(ctx context.Context, key string) error {
	delete(f.data, key)
	return nil
}

func (f *fakeCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	var v int64
	if b, ok := f.data[key]; ok {
		_ = json.Unmarshal(b, &v)
	}
	v += delta
	b, _ := json.Marshal(v)
	f.data[key] = b
	return v, nil
}

func (f *fakeCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return nil
}

func (f *fakeCache) Ping(ctx context.Context) error { return nil }

func (f *fakeCache) Close() error { return nil }

type fakeRoleRepo struct {
	byCode map[string]*domain.Role
	perms  map[string][]string
}

func newFakeRoleRepo(roles ...*domain.Role) *fakeRoleRepo {
	f := &fakeRoleRepo{
		byCode: map[string]*domain.Role{},
		perms:  map[string][]string{},
	}
	for _, r := range roles {
		f.byCode[r.Code] = r
	}
	return f
}

func (f *fakeRoleRepo) Save(ctx context.Context, r *domain.Role) error {
	f.byCode[r.Code] = r
	return nil
}

func (f *fakeRoleRepo) FindByCode(ctx context.Context, code string) (*domain.Role, error) {
	return f.byCode[code], nil
}

func (f *fakeRoleRepo) List(ctx context.Context) ([]*domain.Role, error) {
	out := make([]*domain.Role, 0, len(f.byCode))
	for _, r := range f.byCode {
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeRoleRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (f *fakeRoleRepo) UpdateName(ctx context.Context, id, name string) error {
	return nil
}

func (f *fakeRoleRepo) SetPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	return nil
}

func (f *fakeRoleRepo) PermissionsFor(ctx context.Context, roleID string) ([]string, error) {
	return f.perms[roleID], nil
}

type fakePermissionRepo struct {
	perms []*domain.Permission
}

func (f *fakePermissionRepo) Save(ctx context.Context, p *domain.Permission) error {
	return nil
}

func (f *fakePermissionRepo) FindByCode(ctx context.Context, code string) (*domain.Permission, error) {
	return nil, nil
}

func (f *fakePermissionRepo) List(ctx context.Context) ([]*domain.Permission, error) {
	return f.perms, nil
}

func (f *fakePermissionRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (f *fakePermissionRepo) Update(ctx context.Context, id, group, name string) error {
	return nil
}

type fakeUserAccessRepo struct {
	roles        []string
	directPerms  []string
	inherited    []string
	loadRolesErr error
}

func (f *fakeUserAccessRepo) AssignRole(ctx context.Context, userID, roleID string) error {
	return nil
}

func (f *fakeUserAccessRepo) RevokeRole(ctx context.Context, userID, roleID string) error {
	return nil
}

func (f *fakeUserAccessRepo) GrantPermission(ctx context.Context, userID, permissionID string) error {
	return nil
}

func (f *fakeUserAccessRepo) RevokePermission(ctx context.Context, userID, permissionID string) error {
	return nil
}

func (f *fakeUserAccessRepo) Roles(ctx context.Context, userID string) ([]string, error) {
	return f.roles, f.loadRolesErr
}

func (f *fakeUserAccessRepo) DirectPermissions(ctx context.Context, userID string) ([]string, error) {
	return f.directPerms, nil
}

func (f *fakeUserAccessRepo) RolePermissionCodes(ctx context.Context, userID string) ([]string, error) {
	return f.inherited, nil
}
