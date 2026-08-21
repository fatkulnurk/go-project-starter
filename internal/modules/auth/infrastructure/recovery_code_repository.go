// Package infrastructure implements the auth module's domain repositories
// with database/sql. SQL is written with '?' placeholders and rebound for
// postgres, so the same code runs on MySQL and PostgreSQL.
package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// RecoveryCodeRepository implements domain.RecoveryCodeRepository with
// database/sql queries over the user_recovery_codes table.
type RecoveryCodeRepository struct{ base }

// NewRecoveryCodeRepository builds a recovery-code repository bound to the
// read/write pools, driver and app timezone.
func NewRecoveryCodeRepository(readDB, writeDB *sql.DB, driver string, loc *time.Location) *RecoveryCodeRepository {
	return &RecoveryCodeRepository{base{readDB: readDB, writeDB: writeDB, driver: driver, loc: loc}}
}

// SaveAll implements domain.RecoveryCodeRepository. The code hashes are
// (user_id, code_hash) primary keys, so re-issuing a set after MFA re-activation
// must first clear the previous ones.
func (r *RecoveryCodeRepository) SaveAll(ctx context.Context, userID string, codeHashes []string) error {
	tx, err := r.w().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	const del = `DELETE FROM user_recovery_codes WHERE user_id = ?`
	if _, err := tx.ExecContext(ctx, r.q(del), userID); err != nil {
		return err
	}
	const ins = `INSERT INTO user_recovery_codes (user_id, code_hash, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)`
	for _, hash := range codeHashes {
		if _, err := tx.ExecContext(ctx, r.q(ins), userID, hash); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Consume implements domain.RecoveryCodeRepository. Only one concurrent
// request can win the single-use update, so a code cannot be reused.
func (r *RecoveryCodeRepository) Consume(ctx context.Context, userID, codeHash string) (bool, error) {
	now := r.now()
	res, err := r.w().ExecContext(ctx, r.q(`UPDATE user_recovery_codes SET used_at = ? WHERE user_id = ? AND code_hash = ? AND used_at IS NULL`), now, userID, codeHash)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// DeleteAll implements domain.RecoveryCodeRepository, removing every recovery
// code stored for the user.
func (r *RecoveryCodeRepository) DeleteAll(ctx context.Context, userID string) error {
	_, err := r.w().ExecContext(ctx, r.q(`DELETE FROM user_recovery_codes WHERE user_id = ?`), userID)
	return err
}

var _ domain.RecoveryCodeRepository = (*RecoveryCodeRepository)(nil)
