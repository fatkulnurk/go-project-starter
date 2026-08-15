package commands

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/application/id"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

type fakeGenerator struct{ v string }

func (f fakeGenerator) New() string { return f.v }

func init() {
	id.SetDefault(fakeGenerator{v: "fixed-id"})
}

type fakeAuditor struct {
	entries []audit.Entry
}

func (f *fakeAuditor) Record(ctx context.Context, entry audit.Entry) error {
	f.entries = append(f.entries, entry)
	return nil
}

type fakeBumper struct{ bumped int }

func (f *fakeBumper) Bump(ctx context.Context) error {
	f.bumped++
	return nil
}

type fakeRoleRepo struct {
	byCode  map[string]*domain.Role
	deleted []string
	renamed map[string]string
	perms   map[string][]string
}

func newFakeRoleRepo(roles ...*domain.Role) *fakeRoleRepo {
	f := &fakeRoleRepo{
		byCode:  map[string]*domain.Role{},
		renamed: map[string]string{},
		perms:   map[string][]string{},
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
	f.deleted = append(f.deleted, id)
	for code, r := range f.byCode {
		if r.ID == id {
			delete(f.byCode, code)
		}
	}
	return nil
}

func (f *fakeRoleRepo) UpdateName(ctx context.Context, id, name string) error {
	f.renamed[id] = name
	for code, r := range f.byCode {
		if r.ID == id {
			delete(f.byCode, code)
			r.Name = name
			f.byCode[r.Code] = r
		}
	}
	return nil
}

func (f *fakeRoleRepo) SetPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	return nil
}

func (f *fakeRoleRepo) PermissionsFor(ctx context.Context, roleID string) ([]string, error) {
	return f.perms[roleID], nil
}

type fakePermissionRepo struct {
	byCode  map[string]*domain.Permission
	deleted []string
	updated map[string]*domain.Permission
}

func newFakePermissionRepo(perms ...*domain.Permission) *fakePermissionRepo {
	f := &fakePermissionRepo{
		byCode:  map[string]*domain.Permission{},
		updated: map[string]*domain.Permission{},
	}
	for _, p := range perms {
		f.byCode[p.Code] = p
	}
	return f
}

func (f *fakePermissionRepo) Save(ctx context.Context, p *domain.Permission) error {
	f.byCode[p.Code] = p
	return nil
}

func (f *fakePermissionRepo) FindByCode(ctx context.Context, code string) (*domain.Permission, error) {
	return f.byCode[code], nil
}

func (f *fakePermissionRepo) List(ctx context.Context) ([]*domain.Permission, error) {
	out := make([]*domain.Permission, 0, len(f.byCode))
	for _, p := range f.byCode {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakePermissionRepo) Delete(ctx context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	for code, p := range f.byCode {
		if p.ID == id {
			delete(f.byCode, code)
		}
	}
	return nil
}

func (f *fakePermissionRepo) Update(ctx context.Context, id, group, name string) error {
	for code, p := range f.byCode {
		if p.ID == id {
			delete(f.byCode, code)
			p.Group = group
			p.Name = name
			f.byCode[p.Code] = p
			f.updated[id] = p
		}
	}
	return nil
}

type fakeUserAccessRepo struct {
	roles       map[string][]string
	directPerms map[string][]string
}

func (f *fakeUserAccessRepo) AssignRole(ctx context.Context, userID, roleID string) error {
	f.roles[userID] = append(f.roles[userID], roleID)
	return nil
}

func (f *fakeUserAccessRepo) RevokeRole(ctx context.Context, userID, roleID string) error {
	out := f.roles[userID][:0]
	for _, id := range f.roles[userID] {
		if id != roleID {
			out = append(out, id)
		}
	}
	f.roles[userID] = out
	return nil
}

func (f *fakeUserAccessRepo) GrantPermission(ctx context.Context, userID, permissionID string) error {
	f.directPerms[userID] = append(f.directPerms[userID], permissionID)
	return nil
}

func (f *fakeUserAccessRepo) RevokePermission(ctx context.Context, userID, permissionID string) error {
	out := f.directPerms[userID][:0]
	for _, id := range f.directPerms[userID] {
		if id != permissionID {
			out = append(out, id)
		}
	}
	f.directPerms[userID] = out
	return nil
}

func (f *fakeUserAccessRepo) Roles(ctx context.Context, userID string) ([]string, error) {
	return f.roles[userID], nil
}

func (f *fakeUserAccessRepo) DirectPermissions(ctx context.Context, userID string) ([]string, error) {
	return f.directPerms[userID], nil
}

func (f *fakeUserAccessRepo) RolePermissionCodes(ctx context.Context, userID string) ([]string, error) {
	return nil, nil
}
