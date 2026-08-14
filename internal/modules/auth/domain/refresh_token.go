package domain

import "time"

// RefreshToken is an opaque, revocable, rotating long-lived credential.
// Only its SHA-256 hash is stored; the raw token is shown once at issue time.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewRefreshToken builds a refresh token from its raw value at now.
func NewRefreshToken(userID, rawToken string, ttl time.Duration, now time.Time) *RefreshToken {
	now = now.UTC()
	return &RefreshToken{
		ID:        newID(),
		UserID:    userID,
		TokenHash: HashSecret(rawToken),
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsExpired reports whether the token is past its expiry.
func (t *RefreshToken) IsExpired(now time.Time) bool {
	return now.UTC().After(t.ExpiresAt)
}

// IsRevoked reports whether the token was revoked.
func (t *RefreshToken) IsRevoked() bool { return t.RevokedAt != nil }

// Revoke marks the token as revoked at now.
func (t *RefreshToken) Revoke(now time.Time) {
	now = now.UTC()
	t.RevokedAt = &now
	t.UpdatedAt = now
}
