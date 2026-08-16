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

// RoleRepository implements domain.RoleRepository using database/sql. SQL uses
// '?' placeholders, rebound for postgres.
type RoleRepository struct{ base }

// NewRoleRepository builds a role repository. driver selects the placeholder
// dialect (mysql or postgres) used for rebinding queries.
func NewRoleRepository(db *sql.DB, driver string) *RoleRepository {
	return &RoleRepository{base{db: db, driver: driver}}
}

// Save implements domain.RoleRepository. It inserts the role row and returns
// an error if a role with the same id already exists.
func (r *RoleRepository) Save(ctx context.Context, role *domain.Role) error {
	_, err := r.db.ExecContext(ctx, r.q(`INSERT INTO roles (id, code, name) VALUES (?, ?, ?)`), role.ID, role.Code, role.Name)
	return err
}

// FindByCode implements domain.RoleRepository. It returns the role with code,
// or nil when no such role exists.
func (r *RoleRepository) FindByCode(ctx context.Context, code string) (*domain.Role, error) {
	row := r.db.QueryRowContext(ctx, r.q(`SELECT id, code, name FROM roles WHERE code = ?`), code)
	return scanRole(row)
}

// List implements domain.RoleRepository. It returns all roles ordered by
// creation time ascending.
func (r *RoleRepository) List(ctx context.Context) ([]*domain.Role, error) {
	rows, err := r.db.QueryContext(ctx, r.q(`SELECT id, code, name FROM roles ORDER BY created_at ASC`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.Code, &role.Name); err != nil {
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

// UpdateName implements domain.RoleRepository. It renames the role's display
// label and touches updated_at; the role's users and permission links are
// untouched.
func (r *RoleRepository) UpdateName(ctx context.Context, id, name string) error {
	_, err := r.db.ExecContext(ctx, r.q(`UPDATE roles SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`), name, id)
	return err
}

// SetPermissions implements domain.RoleRepository. It deletes the role's
// current links and inserts the given set inside a single transaction, so the
// replacement is atomic and leaves no partial state on failure.
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

// PermissionsFor implements domain.RoleRepository. It returns the permission
// codes of one role, ordered by code, via a join across role_permissions.
func (r *RoleRepository) PermissionsFor(ctx context.Context, roleID string) ([]string, error) {
	return r.scanStrings(ctx, `
		SELECT p.code
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = ?
		ORDER BY p.code`, roleID)
}

// PermissionsForAll implements domain.RoleRepository. It loads every role's
// permission codes in one query, keyed by role id, so callers can build list
// responses without an N+1 query pattern.
func (r *RoleRepository) PermissionsForAll(ctx context.Context) (map[string][]string, error) {
	rows, err := r.db.QueryContext(ctx, r.q(`
		SELECT rp.role_id, p.code
		FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		ORDER BY p.code`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]string)
	for rows.Next() {
		var roleID, code string
		if err := rows.Scan(&roleID, &code); err != nil {
			return nil, err
		}
		out[roleID] = append(out[roleID], code)
	}
	return out, rows.Err()
}

func scanRole(row *sql.Row) (*domain.Role, error) {
	var role domain.Role
	err := row.Scan(&role.ID, &role.Code, &role.Name)
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

// PermissionRepository implements domain.PermissionRepository using
// database/sql. SQL uses '?' placeholders, rebound for postgres.
type PermissionRepository struct{ base }

// NewPermissionRepository builds a permission repository. driver selects the
// placeholder dialect (mysql or postgres) used for rebinding queries.
func NewPermissionRepository(db *sql.DB, driver string) *PermissionRepository {
	return &PermissionRepository{base{db: db, driver: driver}}
}

// Save implements domain.PermissionRepository. It inserts the permission row
// and returns an error if a permission with the same id already exists.
func (r *PermissionRepository) Save(ctx context.Context, p *domain.Permission) error {
	_, err := r.db.ExecContext(ctx, r.q(`INSERT INTO permissions (id, code, group_name, name) VALUES (?, ?, ?, ?)`), p.ID, p.Code, p.Group, p.Name)
	return err
}

// FindByCode implements domain.PermissionRepository. It returns the permission
// with code, or nil when no such permission exists.
func (r *PermissionRepository) FindByCode(ctx context.Context, code string) (*domain.Permission, error) {
	row := r.db.QueryRowContext(ctx, r.q(`SELECT id, code, group_name, name FROM permissions WHERE code = ?`), code)
	var p domain.Permission
	err := row.Scan(&p.ID, &p.Code, &p.Group, &p.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List implements domain.PermissionRepository. Results are ordered by group
// then code so grouped UI rendering needs no extra sorting.
func (r *PermissionRepository) List(ctx context.Context) ([]*domain.Permission, error) {
	rows, err := r.db.QueryContext(ctx, r.q(`SELECT id, code, group_name, name FROM permissions ORDER BY group_name ASC, code ASC`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Group, &p.Name); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// Delete implements domain.PermissionRepository. Links are removed by ON
// DELETE CASCADE.
func (r *PermissionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, r.q(`DELETE FROM permissions WHERE id = ?`), id)
	return err
}

// Update implements domain.PermissionRepository. It renames the display group
// and label, keeping the code and its links.
func (r *PermissionRepository) Update(ctx context.Context, id, group, name string) error {
	_, err := r.db.ExecContext(ctx, r.q(`UPDATE permissions SET group_name = ?, name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`), group, name, id)
	return err
}

// ---------------------------------------------------------------------------
// User access (user_roles, user_permissions)
// ---------------------------------------------------------------------------

// UserAccessRepository implements domain.UserAccessRepository using
// database/sql. Grants are idempotent (INSERT IGNORE / ON CONFLICT DO NOTHING).
type UserAccessRepository struct{ base }

// NewUserAccessRepository builds a user access repository. driver selects the
// placeholder dialect (mysql or postgres) used for rebinding queries.
func NewUserAccessRepository(db *sql.DB, driver string) *UserAccessRepository {
	return &UserAccessRepository{base{db: db, driver: driver}}
}

// AssignRole implements domain.UserAccessRepository. It inserts the link using
// an idempotent insert, so assigning an already-held role is a no-op.
func (r *UserAccessRepository) AssignRole(ctx context.Context, userID, roleID string) error {
	_, err := r.db.ExecContext(ctx, r.q(r.insertIgnore("user_roles", "user_id, role_id", "?, ?")), userID, roleID)
	return err
}

// RevokeRole implements domain.UserAccessRepository. It removes the link if it
// exists and does not error otherwise.
func (r *UserAccessRepository) RevokeRole(ctx context.Context, userID, roleID string) error {
	_, err := r.db.ExecContext(ctx, r.q(`DELETE FROM user_roles WHERE user_id = ? AND role_id = ?`), userID, roleID)
	return err
}

// GrantPermission implements domain.UserAccessRepository. It inserts the link
// using an idempotent insert, so granting an already-held permission is a
// no-op.
func (r *UserAccessRepository) GrantPermission(ctx context.Context, userID, permissionID string) error {
	_, err := r.db.ExecContext(ctx, r.q(r.insertIgnore("user_permissions", "user_id, permission_id", "?, ?")), userID, permissionID)
	return err
}

// RevokePermission implements domain.UserAccessRepository. It removes the link
// if it exists and does not error otherwise.
func (r *UserAccessRepository) RevokePermission(ctx context.Context, userID, permissionID string) error {
	_, err := r.db.ExecContext(ctx, r.q(`DELETE FROM user_permissions WHERE user_id = ? AND permission_id = ?`), userID, permissionID)
	return err
}

// Roles implements domain.UserAccessRepository. It returns the role codes of
// a user, ordered by code, via a join across the user_roles link table.
func (r *UserAccessRepository) Roles(ctx context.Context, userID string) ([]string, error) {
	return r.scanStrings(ctx, `
		SELECT r.code
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = ?
		ORDER BY r.code`, userID)
}

// CountUsersWithRole implements domain.UserAccessRepository. It counts the
// users currently holding the role, used to protect the last super_admin from
// being revoked.
func (r *UserAccessRepository) CountUsersWithRole(ctx context.Context, roleID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, r.q(`SELECT COUNT(*) FROM user_roles WHERE role_id = ?`), roleID).Scan(&n)
	return n, err
}

// DirectPermissions implements domain.UserAccessRepository. It returns
// permission codes.
func (r *UserAccessRepository) DirectPermissions(ctx context.Context, userID string) ([]string, error) {
	return r.scanStrings(ctx, `
		SELECT p.code
		FROM permissions p
		JOIN user_permissions up ON up.permission_id = p.id
		WHERE up.user_id = ?
		ORDER BY p.code`, userID)
}

// RolePermissionCodes implements domain.UserAccessRepository. It returns the
// permission codes inherited through the user's roles.
func (r *UserAccessRepository) RolePermissionCodes(ctx context.Context, userID string) ([]string, error) {
	return r.scanStrings(ctx, `
		SELECT DISTINCT p.code
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = ?
		ORDER BY p.code`, userID)
}
