// Package infrastructure implements the media module's domain repository with
// database/sql. SQL uses '?' placeholders, rebound for postgres.
package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fatkulnurk/go-project-starter/internal/modules/media/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
)

// MediaRepository implements domain.MediaRepository.
type MediaRepository struct {
	db     *sql.DB
	driver string
}

// NewMediaRepository builds a media repository.
func NewMediaRepository(db *sql.DB, driver string) *MediaRepository {
	return &MediaRepository{db: db, driver: driver}
}

func (r *MediaRepository) q(query string) string { return database.Rebind(query, r.driver) }

const mediaColumns = `id, model_type, model_id, collection_name, name, file_name, mime_type, disk, size, created_at, updated_at`

// Save implements domain.MediaRepository.
func (r *MediaRepository) Save(ctx context.Context, m *domain.Media) error {
	_, err := r.db.ExecContext(ctx, r.q(`
		INSERT INTO media (`+mediaColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		m.ID, m.ModelType, m.ModelID, m.CollectionName, m.Name, m.FileName,
		m.MimeType, m.Disk, m.Size, m.CreatedAt, m.UpdatedAt)
	return err
}

// FindByID implements domain.MediaRepository.
func (r *MediaRepository) FindByID(ctx context.Context, id string) (*domain.Media, error) {
	row := r.db.QueryRowContext(ctx, r.q(`SELECT `+mediaColumns+` FROM media WHERE id = ?`), id)
	return scanMedia(row)
}

// ListByModel implements domain.MediaRepository.
func (r *MediaRepository) ListByModel(ctx context.Context, modelType, modelID, collection string) ([]*domain.Media, error) {
	query := `SELECT ` + mediaColumns + ` FROM media WHERE model_type = ? AND model_id = ?`
	args := []any{modelType, modelID}
	if collection != "" {
		query += ` AND collection_name = ?`
		args = append(args, collection)
	}
	query += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, r.q(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Media
	for rows.Next() {
		m, err := scanMediaRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Delete implements domain.MediaRepository.
func (r *MediaRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, r.q(`DELETE FROM media WHERE id = ?`), id)
	return err
}

func scanMedia(row *sql.Row) (*domain.Media, error) {
	m, err := scanMediaRows(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMediaRows(row rowScanner) (*domain.Media, error) {
	var m domain.Media
	err := row.Scan(
		&m.ID, &m.ModelType, &m.ModelID, &m.CollectionName, &m.Name, &m.FileName,
		&m.MimeType, &m.Disk, &m.Size, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
