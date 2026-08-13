// Package infrastructure implements the RBAC module's domain repositories
// with database/sql. SQL uses '?' placeholders, rebound for postgres.
package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
)

// base carries the shared pool and driver used by every repository type.
type base struct {
	db     *sql.DB
	driver string
}

func (b base) q(query string) string { return database.Rebind(query, b.driver) }

// insertIgnore returns an INSERT that ignores duplicate keys (MySQL) or does
// nothing on conflict (PostgreSQL), keeping role/permission grants idempotent.
func (b base) insertIgnore(table, columns, placeholders string) string {
	insert := "INSERT INTO " + table + " (" + columns + ") VALUES (" + placeholders + ")"
	if b.driver == config.DriverPostgres {
		return insert + " ON CONFLICT DO NOTHING"
	}
	return "INSERT IGNORE INTO " + table + " (" + columns + ") VALUES (" + placeholders + ")"
}

func (b base) scanStrings(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := b.db.QueryContext(ctx, b.q(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Roles
// ---------------------------------------------------------------------------

// RoleRepository implements domain.RoleRepository.
type RoleRepository struct{ base }

// NewRoleRepository builds a role repository.
func NewRoleRepository(db *sql.DB, driver string) *RoleRepository {
	return &RoleRepository{base{db: db, driver: driver}}
}

// Save implements domain.RoleRepository.
func (r *RoleRepository) Save(ctx context.Context, role *domain.Role) error {
	_, err := r.db.ExecContext(ctx, r.q(`INSERT INTO roles (id, name) VALUES (?, ?)`), role.ID, role.Name)
	return err
}

// FindByName implements domain.RoleRepository.
func (r *RoleRepository) FindByName(ctx context.Context, name string) (*domain.Role, error) {
	row := r.db.QueryRowContext(ctx, r.q(`SELECT id, name FROM roles WHERE name = ?`), name)
	return scanRole(row)
}

// List implements domain.RoleRepository.
func (r *RoleRepository) List(ctx context.Context) ([]*domain.Role, error) {
	rows, err := r.db.QueryContext(ctx, r.q(`SELECT id, name FROM roles ORDER BY created_at ASC`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.Name); err != nil {
			return nil, err
		}
		out = append(out, &role)
	}
	return out, rows.Err()
}

// Delete implements domain.RoleRepository. Links are removed by ON DELETE
// CASCADE.
func (r *RoleRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, r.q(`DELETE FROM roles WHERE id = ?`), id)
	return err
}

// SetPermissions implements domain.RoleRepository.
func (r *RoleRepository) SetPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, r.q(`DELETE FROM role_permissions WHERE role_id = ?`), roleID); err != nil {
		return err
	}
	for _, pid := range permissionIDs {
		if _, err := tx.ExecContext(ctx, r.q(`INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)`), roleID, pid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// PermissionsFor implements domain.RoleRepository.
func (r *RoleRepository) PermissionsFor(ctx context.Context, roleID string) ([]string, error) {
	return r.scanStrings(ctx, `
		SELECT p.name
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = ?
		ORDER BY p.name`, roleID)
}

func scanRole(row *sql.Row) (*domain.Role, error) {
	var role domain.Role
	err := row.Scan(&role.ID, &role.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// ---------------------------------------------------------------------------
// Permissions
// ---------------------------------------------------------------------------

// PermissionRepository implements domain.PermissionRepository.
type PermissionRepository struct{ base }

// NewPermissionRepository builds a permission repository.
func NewPermissionRepository(db *sql.DB, driver string) *PermissionRepository {
	return &PermissionRepository{base{db: db, driver: driver}}
}

// Save implements domain.PermissionRepository.
func (r *PermissionRepository) Save(ctx context.Context, p *domain.Permission) error {
	_, err := r.db.ExecContext(ctx, r.q(`INSERT INTO permissions (id, name) VALUES (?, ?)`), p.ID, p.Name)
	return err
}

// FindByName implements domain.PermissionRepository.
func (r *PermissionRepository) FindByName(ctx context.Context, name string) (*domain.Permission, error) {
	row := r.db.QueryRowContext(ctx, r.q(`SELECT id, name FROM permissions WHERE name = ?`), name)
	var p domain.Permission
	err := row.Scan(&p.ID, &p.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List implements domain.PermissionRepository.
func (r *PermissionRepository) List(ctx context.Context) ([]*domain.Permission, error) {
	rows, err := r.db.QueryContext(ctx, r.q(`SELECT id, name FROM permissions ORDER BY name ASC`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// User access (user_roles, user_permissions)
// ---------------------------------------------------------------------------

// UserAccessRepository implements domain.UserAccessRepository.
type UserAccessRepository struct{ base }

// NewUserAccessRepository builds a user access repository.
func NewUserAccessRepository(db *sql.DB, driver string) *UserAccessRepository {
	return &UserAccessRepository{base{db: db, driver: driver}}
}

// AssignRole implements domain.UserAccessRepository.
func (r *UserAccessRepository) AssignRole(ctx context.Context, userID, roleID string) error {
	_, err := r.db.ExecContext(ctx, r.q(r.insertIgnore("user_roles", "user_id, role_id", "?, ?")), userID, roleID)
	return err
}

// RevokeRole implements domain.UserAccessRepository.
func (r *UserAccessRepository) RevokeRole(ctx context.Context, userID, roleID string) error {
	_, err := r.db.ExecContext(ctx, r.q(`DELETE FROM user_roles WHERE user_id = ? AND role_id = ?`), userID, roleID)
	return err
}

// GrantPermission implements domain.UserAccessRepository.
func (r *UserAccessRepository) GrantPermission(ctx context.Context, userID, permissionID string) error {
	_, err := r.db.ExecContext(ctx, r.q(r.insertIgnore("user_permissions", "user_id, permission_id", "?, ?")), userID, permissionID)
	return err
}

// RevokePermission implements domain.UserAccessRepository.
func (r *UserAccessRepository) RevokePermission(ctx context.Context, userID, permissionID string) error {
	_, err := r.db.ExecContext(ctx, r.q(`DELETE FROM user_permissions WHERE user_id = ? AND permission_id = ?`), userID, permissionID)
	return err
}

// Roles implements domain.UserAccessRepository.
func (r *UserAccessRepository) Roles(ctx context.Context, userID string) ([]string, error) {
	return r.scanStrings(ctx, `
		SELECT r.name
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = ?
		ORDER BY r.name`, userID)
}

// DirectPermissions implements domain.UserAccessRepository.
func (r *UserAccessRepository) DirectPermissions(ctx context.Context, userID string) ([]string, error) {
	return r.scanStrings(ctx, `
		SELECT p.name
		FROM permissions p
		JOIN user_permissions up ON up.permission_id = p.id
		WHERE up.user_id = ?
		ORDER BY p.name`, userID)
}

// RolePermissionNames implements domain.UserAccessRepository.
func (r *UserAccessRepository) RolePermissionNames(ctx context.Context, userID string) ([]string, error) {
	return r.scanStrings(ctx, `
		SELECT DISTINCT p.name
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = ?
		ORDER BY p.name`, userID)
}
