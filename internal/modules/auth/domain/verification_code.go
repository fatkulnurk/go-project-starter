package domain

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

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
// value (already validated by the caller). The secret is stored hashed; OTPs
// (low entropy) use bcrypt so an offline database leak cannot be brute-forced,
// while magic links (high-entropy opaque tokens) use a fast SHA-256 digest.
func NewVerificationCode(userID string, channel Channel, purpose Purpose, raw string, ttl time.Duration, now time.Time) (*VerificationCode, error) {
	now = now.UTC()
	codeHash, err := hashCode(raw, purpose)
	if err != nil {
		return nil, err
	}
	return &VerificationCode{
		ID:        newID(),
		UserID:    userID,
		Channel:   channel,
		Purpose:   purpose,
		CodeHash:  codeHash,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
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
	if c.Purpose == PurposeMagicLink {
		return HashSecret(raw) == c.CodeHash
	}
	return bcrypt.CompareHashAndPassword([]byte(c.CodeHash), []byte(raw)) == nil
}

// hashCode hashes the raw secret for storage: bcrypt for OTPs (low entropy),
// fast SHA-256 for magic links (high-entropy tokens kept searchable by hash).
func hashCode(raw string, purpose Purpose) (string, error) {
	if purpose == PurposeMagicLink {
		return HashSecret(raw), nil
	}
	b, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
