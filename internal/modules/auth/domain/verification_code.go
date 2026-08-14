package domain

import "time"

// Channel identifies the delivery channel of a verification code.
type Channel string

// Channels.
const (
	ChannelEmail Channel = "email"
	ChannelPhone Channel = "phone"
)

// Purpose identifies what the code/token is used for.
type Purpose string

// Purposes.
const (
	PurposeVerify    Purpose = "verify"
	PurposeReset     Purpose = "reset"
	PurposeMagicLink Purpose = "magic_link"
)

// VerificationCode is a single-use, expiring, attempt-limited code or token.
type VerificationCode struct {
	ID         string
	UserID     string
	Channel    Channel
	Purpose    Purpose
	CodeHash   string
	Attempts   int
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewVerificationCode builds a code for a user/channel/purpose from its raw
// value (already validated by the caller).
func NewVerificationCode(userID string, channel Channel, purpose Purpose, raw string, ttl time.Duration, now time.Time) *VerificationCode {
	now = now.UTC()
	return &VerificationCode{
		ID:        newID(),
		UserID:    userID,
		Channel:   channel,
		Purpose:   purpose,
		CodeHash:  HashSecret(raw),
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsExpired reports whether the code is past its expiry.
func (c *VerificationCode) IsExpired(now time.Time) bool {
	return now.UTC().After(c.ExpiresAt)
}

// IsConsumed reports whether the code was already used.
func (c *VerificationCode) IsConsumed() bool { return c.ConsumedAt != nil }

// Consume marks the code as used at now.
func (c *VerificationCode) Consume(now time.Time) {
	now = now.UTC()
	c.ConsumedAt = &now
	c.UpdatedAt = now
}

// Matches reports whether raw equals the stored secret.
func (c *VerificationCode) Matches(raw string) bool {
	return HashSecret(raw) == c.CodeHash
}
