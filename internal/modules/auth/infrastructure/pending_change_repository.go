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

// PendingContactChangeRepository implements domain.PendingContactChangeRepository
// with database/sql queries over the pending_contact_changes table.
type PendingContactChangeRepository struct{ base }

// NewPendingContactChangeRepository builds a pending-change repository bound
// to the shared pool, driver and app timezone.
func NewPendingContactChangeRepository(db *sql.DB, driver string, loc *time.Location) *PendingContactChangeRepository {
	return &PendingContactChangeRepository{base{db: db, driver: driver, loc: loc}}
}

const pendingChangeColumns = `id, user_id, channel, old_value, new_value, status, applied_at, created_at, updated_at`

// Save implements domain.PendingContactChangeRepository, inserting the pending
// change with its current status.
func (r *PendingContactChangeRepository) Save(ctx context.Context, p *domain.PendingContactChange) error {
	_, err := r.db.ExecContext(ctx, r.q(`
		INSERT INTO pending_contact_changes (`+pendingChangeColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		p.ID, p.UserID, string(p.Channel), p.OldValue, p.NewValue,
		string(p.Status), nullTime(p.AppliedAt), p.CreatedAt, p.UpdatedAt)
	return err
}

// FindPendingByNewValue implements domain.PendingContactChangeRepository,
// returning the newest pending change matching the channel and value, or
// (nil, nil) when none exists.
func (r *PendingContactChangeRepository) FindPendingByNewValue(ctx context.Context, channel domain.Channel, newValue string) (*domain.PendingContactChange, error) {
	return r.scanPendingChange(ctx, `
		SELECT `+pendingChangeColumns+` FROM pending_contact_changes
		WHERE channel = ? AND new_value = ? AND status = 'pending'
		ORDER BY created_at DESC LIMIT 1`,
		string(channel), newValue)
}

// MarkApplied implements domain.PendingContactChangeRepository, flipping a
// still-pending change to applied at appliedAt.
func (r *PendingContactChangeRepository) MarkApplied(ctx context.Context, id string, appliedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, r.q(`
		UPDATE pending_contact_changes SET status = 'applied', applied_at = ?, updated_at = ?
		WHERE id = ? AND status = 'pending'`),
		appliedAt, r.now(), id)
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
