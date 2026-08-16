package domain

import "time"

// RefreshToken is an opaque, revocable, rotating long-lived credential.
// Only its SHA-256 hash is stored; the raw token is shown once at issue time.
// FamilyID groups rotations of one login session and JTI ties the access token
// minted alongside this token, so revoking a session also invalidates the
// access tokens already handed out.
type RefreshToken struct {
	ID        string
	UserID    string
	FamilyID  string
	JTI       string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewRefreshToken builds a refresh token from its raw value at now. familyID
// groups it with prior rotations (empty starts a new session family); jti is
// the access-token id minted together with this token.
func NewRefreshToken(userID, rawToken, familyID, jti string, ttl time.Duration, now time.Time) *RefreshToken {
	now = now.UTC()
	return &RefreshToken{
		ID:        newID(),
		UserID:    userID,
		FamilyID:  familyID,
		JTI:       jti,
		TokenHash: HashSecret(rawToken),
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// RefreshFamily is one login session: the family of refresh tokens produced by
// rotations of a single credential. It is active while at least one of its
// tokens is un-revoked and un-expired.
type RefreshFamily struct {
	ID        string
	CreatedAt time.Time
	LastUsed  time.Time
	ExpiresAt time.Time
}

// IsExpired reports whether the token is past its expiry relative to now.
// The reference time is converted to UTC before comparison.
func (t *RefreshToken) IsExpired(now time.Time) bool {
	return now.UTC().After(t.ExpiresAt)
}

// IsRevoked reports whether the token was revoked, i.e. its RevokedAt
// timestamp is non-nil.
func (t *RefreshToken) IsRevoked() bool { return t.RevokedAt != nil }

// Revoke marks the token as revoked at now and stamps the record as updated.
// Revoked tokens are rejected by the refresh flow.
func (t *RefreshToken) Revoke(now time.Time) {
	now = now.UTC()
	t.RevokedAt = &now
	t.UpdatedAt = now
}
