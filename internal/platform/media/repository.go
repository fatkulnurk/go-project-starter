package media

import (
	"context"
	"database/sql"
	"errors"

	appmedia "github.com/fatkulnurk/go-project-starter/internal/application/media"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
)

// mediaRepository persists media records. It is satisfied by Repository and
// can be faked in tests.
type mediaRepository interface {
	Save(ctx context.Context, m *appmedia.Media) error
	FindByID(ctx context.Context, id string) (*appmedia.Media, error)
	ListByModel(ctx context.Context, modelType, modelID, collection string) ([]*appmedia.Media, error)
	Delete(ctx context.Context, id string) error
}

// Repository persists media records in the media table. SQL uses '?'
// placeholders, rebound for postgres.
type Repository struct {
	db     *sql.DB
	driver string
}

// NewRepository builds a media repository.
func NewRepository(db *sql.DB, driver string) *Repository {
	return &Repository{db: db, driver: driver}
}

var _ mediaRepository = (*Repository)(nil)

func (r *Repository) q(query string) string { return database.Rebind(query, r.driver) }

const mediaColumns = `id, model_type, model_id, collection_name, name, file_name, mime_type, disk, size, created_at, updated_at`

// Save persists a media record.
func (r *Repository) Save(ctx context.Context, m *appmedia.Media) error {
	_, err := r.db.ExecContext(ctx, r.q(`
		INSERT INTO media (`+mediaColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		m.ID, m.ModelType, m.ModelID, m.CollectionName, m.Name, m.FileName,
		m.MimeType, m.Disk, m.Size, m.CreatedAt, m.UpdatedAt)
	return err
}

// FindByID returns a media record, or nil when missing.
func (r *Repository) FindByID(ctx context.Context, id string) (*appmedia.Media, error) {
	row := r.db.QueryRowContext(ctx, r.q(`SELECT `+mediaColumns+` FROM media WHERE id = ?`), id)
	m, err := scanMedia(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

// ListByModel returns the media attached to a model, optionally filtered by
// collection.
func (r *Repository) ListByModel(ctx context.Context, modelType, modelID, collection string) ([]*appmedia.Media, error) {
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

	var out []*appmedia.Media
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Delete removes a media record by id.
func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, r.q(`DELETE FROM media WHERE id = ?`), id)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMedia(row rowScanner) (*appmedia.Media, error) {
	var m appmedia.Media
	err := row.Scan(
		&m.ID, &m.ModelType, &m.ModelID, &m.CollectionName, &m.Name, &m.FileName,
		&m.MimeType, &m.Disk, &m.Size, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
