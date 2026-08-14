// Package audit defines the cross-cutting audit contract: recording who did
// what to which record. Implementations live behind the Auditor interface (SQL,
// file, cloud) so business modules never depend on a storage library.
package audit

import "context"

// Action is the kind of change being recorded.
type Action string

// Actions.
const (
	ActionCreated Action = "created"
	ActionUpdated Action = "updated"
	ActionDeleted Action = "deleted"
)

// ActorType identifies the kind of actor behind a change.
type ActorType string

// Actor types.
const (
	ActorUser   ActorType = "user"
	ActorSystem ActorType = "system"
)

// Actor describes who performed the change.
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

// Auditor records audit entries.
type Auditor interface {
	// Record persists one audit entry.
	Record(ctx context.Context, entry Entry) error
}

type ctxKey struct{}

// WithActor stores the current actor in the context.
func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, ctxKey{}, actor)
}

// ActorFrom returns the actor stored by WithActor, or a zero Actor.
func ActorFrom(ctx context.Context) Actor {
	actor, _ := ctx.Value(ctxKey{}).(Actor)
	return actor
}
