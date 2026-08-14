package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// ---------------------------------------------------------------------------
// Pending contact changes
// ---------------------------------------------------------------------------

// PendingContactChangeRepository implements domain.PendingContactChangeRepository.
type PendingContactChangeRepository struct{ base }

// NewPendingContactChangeRepository builds a pending contact change repository.
func NewPendingContactChangeRepository(db *sql.DB, driver string) *PendingContactChangeRepository {
	return &PendingContactChangeRepository{base{db: db, driver: driver}}
}

const pendingChangeColumns = `id, user_id, channel, old_value, new_value, status, applied_at, created_at, updated_at`

// Save implements domain.PendingContactChangeRepository.
func (r *PendingContactChangeRepository) Save(ctx context.Context, p *domain.PendingContactChange) error {
	_, err := r.db.ExecContext(ctx, r.q(`
		INSERT INTO pending_contact_changes (`+pendingChangeColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		p.ID, p.UserID, string(p.Channel), p.OldValue, p.NewValue,
		string(p.Status), nullTime(p.AppliedAt), p.CreatedAt, p.UpdatedAt)
	return err
}

// FindPendingByNewValue implements domain.PendingContactChangeRepository.
func (r *PendingContactChangeRepository) FindPendingByNewValue(ctx context.Context, channel domain.Channel, newValue string) (*domain.PendingContactChange, error) {
	return r.scanPendingChange(ctx, `
		SELECT `+pendingChangeColumns+` FROM pending_contact_changes
		WHERE channel = ? AND new_value = ? AND status = 'pending'
		ORDER BY created_at DESC LIMIT 1`,
		string(channel), newValue)
}

// MarkApplied implements domain.PendingContactChangeRepository.
func (r *PendingContactChangeRepository) MarkApplied(ctx context.Context, id string, appliedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, r.q(`
		UPDATE pending_contact_changes SET status = 'applied', applied_at = ?, updated_at = ?
		WHERE id = ? AND status = 'pending'`),
		appliedAt, time.Now().UTC(), id)
	return err
}

func (r *PendingContactChangeRepository) scanPendingChange(ctx context.Context, query string, args ...any) (*domain.PendingContactChange, error) {
	row := r.db.QueryRowContext(ctx, r.q(query), args...)
	var p domain.PendingContactChange
	var appliedAt sql.NullTime
	var channel, status string
	err := row.Scan(&p.ID, &p.UserID, &channel, &p.OldValue, &p.NewValue,
		&status, &appliedAt, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Channel = domain.Channel(channel)
	p.Status = domain.PendingContactChangeStatus(status)
	if appliedAt.Valid {
		v := appliedAt.Time
		p.AppliedAt = &v
	}
	return &p, nil
}
