// Package audit provides a SQL-backed Recorder implementation writing to the
// audit_logs table via database/sql.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/application/id"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
)

// SQLRecorder writes audit entries to the audit_logs table.
// It is built from a shared *sql.DB plus a driver name so every query can be
// rebound to the target dialect.
type SQLRecorder struct {
	db     *sql.DB
	driver string
	loc    *time.Location
}

// New builds a SQL-backed Recorder for the given pool.
// driver selects the placeholder dialect used by database.Rebind; loc sets
// the timezone for timestamps and falls back to UTC when nil.
func New(db *sql.DB, driver string, loc *time.Location) *SQLRecorder {
	return &SQLRecorder{db: db, driver: driver, loc: loc}
}

// now returns the current time in the app timezone (UTC when unset).
func (a *SQLRecorder) now() time.Time {
	if a.loc == nil {
		return time.Now().UTC()
	}
	return time.Now().In(a.loc)
}

// Record inserts one audit entry.
// OldValues and NewValues are stored as JSON, and an entry whose actor has no
// type is recorded as the system actor. It returns the INSERT error, if any.
func (a *SQLRecorder) Record(ctx context.Context, entry audit.Entry) error {
	oldJSON, err := marshalJSON(entry.OldValues)
	if err != nil {
		return err
	}
	newJSON, err := marshalJSON(entry.NewValues)
	if err != nil {
		return err
	}
	// A zero actor (background workers, internal calls) is recorded as the
	// system so actor_type keeps its documented values instead of inserting "".
	if entry.Actor.Type == "" {
		entry.Actor.Type = audit.ActorSystem
	}
	const q = `INSERT INTO audit_logs (id, subject_type, subject_id, action, old_values, new_values, actor_type, actor_id, ip_address, user_agent, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := a.now()
	_, err = a.db.ExecContext(ctx, database.Rebind(q, a.driver),
		newID(), entry.SubjectType, entry.SubjectID, string(entry.Action),
		oldJSON, newJSON, string(entry.Actor.Type), nullStr(entry.Actor.ID),
		nullStr(entry.Actor.IPAddress), nullStr(entry.Actor.UserAgent),
		now, now,
	)
	return err
}

func marshalJSON(v map[string]any) (any, error) {
	if len(v) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func newID() string { return id.New() }
