// Package audit defines the cross-cutting audit contract: recording who did
// what to which record. Implementations live behind the Recorder interface
// (SQL, file, cloud) so business modules never depend on a storage library.
package audit

import (
	"context"
	"log/slog"
)

// RecordBestEffort persists an audit entry, logging the failure when it errors
// so a broken audit trail never breaks the business operation yet is never
// silent. Nil recorders are a no-op.
func RecordBestEffort(ctx context.Context, r Recorder, entry Entry) {
	if r == nil {
		return
	}
	if err := r.Record(ctx, entry); err != nil {
		slog.Warn("audit record failed",
			"subject_type", entry.SubjectType,
			"subject_id", entry.SubjectID,
			"action", entry.Action,
			"err", err,
		)
	}
}

// Action is the kind of change being recorded, e.g. created/updated/deleted.
// Values are stored verbatim in the audit_logs.action column.
type Action string

// Actions.
const (
	ActionCreated Action = "created"
	ActionUpdated Action = "updated"
	ActionDeleted Action = "deleted"
)

// ActorType identifies the kind of actor behind a change: a real user or the
// system itself (workers, internal jobs).
type ActorType string

// Actor types.
const (
	ActorUser   ActorType = "user"
	ActorSystem ActorType = "system"
)

// Actor describes who performed the change: the actor type and id plus the
// request's IP address and user agent for human-reviewable audit trails.
type Actor struct {
	Type      ActorType
	ID        string
	IPAddress string
	UserAgent string
}

// Entry is a single audited change: a polymorphic subject plus its context.
// SubjectType/SubjectID identify any record in the system regardless of table.
type Entry struct {
	SubjectType string
	SubjectID   string
	Action      Action
	OldValues   map[string]any
	NewValues   map[string]any
	Actor       Actor
}

// Recorder persists audit entries to an audit trail. The implementation
// (SQL, file, cloud) is chosen in the composition root.
type Recorder interface {
	// Record persists one audit entry. It returns an error when the entry
	// cannot be written; callers typically use RecordBestEffort to avoid
	// breaking the business operation on an audit failure.
	Record(ctx context.Context, entry Entry) error
}

type ctxKey struct{}

// WithActor stores the current actor in the context.
// Middleware uses it so downstream commands can attach the caller to entries.
func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, ctxKey{}, actor)
}

// ActorFrom returns the actor stored by WithActor, or a zero Actor.
// A zero Actor (Type == "") is treated as the system by the SQL recorder.
func ActorFrom(ctx context.Context) Actor {
	actor, _ := ctx.Value(ctxKey{}).(Actor)
	return actor
}
