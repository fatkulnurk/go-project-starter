package domain

import "time"

// PendingContactChangeStatus describes the lifecycle of a contact change
// request.
type PendingContactChangeStatus string

// Pending contact change statuses track a change from request to application.
// A change is either still awaiting confirmation or already applied.
const (
	PendingStatusPending PendingContactChangeStatus = "pending"
	PendingStatusApplied PendingContactChangeStatus = "applied"
)

// PendingContactChange is a requested change of an email or phone number that
// waits for OTP confirmation (via VerifyEmail/VerifyPhone) before it is
// applied to the user. It keeps the old value so a failed confirmation leaves
// the account untouched.
type PendingContactChange struct {
	ID        string
	UserID    string
	Channel   Channel
	OldValue  string
	NewValue  string
	Status    PendingContactChangeStatus
	AppliedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewPendingContactChange builds a pending change for a user/channel. The
// values must already be normalized by the caller.
func NewPendingContactChange(userID string, channel Channel, oldValue, newValue string, now time.Time) *PendingContactChange {
	now = now.UTC()
	return &PendingContactChange{
		ID:        newID(),
		UserID:    userID,
		Channel:   channel,
		OldValue:  oldValue,
		NewValue:  newValue,
		Status:    PendingStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Apply marks the pending change as applied at now, recording the AppliedAt
// timestamp and bumping UpdatedAt.
func (p *PendingContactChange) Apply(now time.Time) {
	now = now.UTC()
	p.Status = PendingStatusApplied
	p.AppliedAt = &now
	p.UpdatedAt = now
}

// IsApplied reports whether the change was already applied, i.e. its status
// is PendingStatusApplied.
func (p *PendingContactChange) IsApplied() bool { return p.Status == PendingStatusApplied }
