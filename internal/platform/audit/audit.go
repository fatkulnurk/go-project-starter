// Package audit provides a SQL-backed Auditor implementation writing to the
// audit_logs table via database/sql.
package audit

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
)

// SQLAuditor writes audit entries to the audit_logs table.
type SQLAuditor struct {
	db     *sql.DB
	driver string
}

// New builds a SQL-backed Auditor for the given pool.
func New(db *sql.DB, driver string) *SQLAuditor {
	return &SQLAuditor{db: db, driver: driver}
}

// Record inserts one audit entry.
func (a *SQLAuditor) Record(ctx context.Context, entry audit.Entry) error {
	oldJSON, err := marshalJSON(entry.OldValues)
	if err != nil {
		return err
	}
	newJSON, err := marshalJSON(entry.NewValues)
	if err != nil {
		return err
	}
	const q = `INSERT INTO audit_logs (id, subject_type, subject_id, action, old_values, new_values, actor_type, actor_id, ip_address, user_agent, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = a.db.ExecContext(ctx, database.Rebind(q, a.driver),
		newID(), entry.SubjectType, entry.SubjectID, string(entry.Action),
		oldJSON, newJSON, string(entry.Actor.Type), nullStr(entry.Actor.ID),
		nullStr(entry.Actor.IPAddress), nullStr(entry.Actor.UserAgent),
		time.Now().UTC(),
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

func newID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("audit: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}
