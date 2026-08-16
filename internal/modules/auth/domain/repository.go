package domain

import (
	"context"
	"time"
)

// UserRepository persists users and resolves them by id or contact address.
// Lookups return (nil, nil) when no user matches.
type UserRepository interface {
	// Save persists a new user. It returns an error when the write fails or
	// the id already exists.
	Save(ctx context.Context, u *User) error
	// FindByID returns the user with the given id, or (nil, nil) when no user
	// matches.
	FindByID(ctx context.Context, id string) (*User, error)
	// FindByEmail returns the user with the given email, or (nil, nil) when
	// none exists. The email must be normalized by the caller.
	FindByEmail(ctx context.Context, email string) (*User, error)
	// FindByPhone returns the user with the given phone, or (nil, nil) when
	// none exists. The phone must be normalized by the caller.
	FindByPhone(ctx context.Context, phone string) (*User, error)
	// Update overwrites the user record with the provided state, including the
	// verification, TOTP and status fields.
	Update(ctx context.Context, u *User) error
}

// RefreshTokenRepository persists refresh tokens and their session families.
// Tokens are stored by hash; lookups return (nil, nil) when nothing matches.
type RefreshTokenRepository interface {
	// Save persists a refresh token. It returns an error when the write fails
	// or the id already exists.
	Save(ctx context.Context, t *RefreshToken) error
	// FindByHash returns the token with the given hash, or (nil, nil) when no
	// token matches.
	FindByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	// RevokeByID revokes the token with the given id, if it is not already
	// revoked.
	RevokeByID(ctx context.Context, id string) error
	// RevokeByIDIfActive revokes the token only if it is still active (not yet
	// revoked) and reports whether it did. It is used for atomic rotation so a
	// token can only be exchanged once even under concurrent requests.
	RevokeByIDIfActive(ctx context.Context, id string) (bool, error)
	// RevokeByUserID revokes every active (un-revoked) token of the user,
	// signing out all of their sessions.
	RevokeByUserID(ctx context.Context, userID string) error
	// RevokeFamily revokes every active token of a session family, signing out
	// every device sharing that session.
	RevokeFamily(ctx context.Context, familyID string) error
	// JtisByFamily returns the access-token ids minted in a session family, used
	// to deny those access tokens when the family is revoked.
	JtisByFamily(ctx context.Context, familyID string) ([]string, error)
	// JtisByUser returns every access-token id ever minted for the user, used
	// to invalidate all access tokens on password change.
	JtisByUser(ctx context.Context, userID string) ([]string, error)
	// ListActiveFamilies returns the user's sessions (families) that still have
	// at least one active token, newest last.
	ListActiveFamilies(ctx context.Context, userID string, now time.Time) ([]RefreshFamily, error)
}

// VerificationCodeRepository persists verification codes and magic links and
// applies their single-use and attempt budgets atomically.
type VerificationCodeRepository interface {
	// Save persists a verification code or magic link. It returns an error
	// when the write fails.
	Save(ctx context.Context, c *VerificationCode) error
	// FindLatestActive returns the newest non-consumed, non-expired code for a
	// user/purpose/channel. Returns nil, nil when none exists.
	FindLatestActive(ctx context.Context, userID string, purpose Purpose, channel Channel) (*VerificationCode, error)
	// FindActiveByHash returns a non-consumed, non-expired code matching the
	// purpose and secret hash. Returns nil, nil when none exists.
	FindActiveByHash(ctx context.Context, purpose Purpose, codeHash string) (*VerificationCode, error)
	// Consume marks the code consumed only if it is still active and reports
	// whether it did. A false result means another request already won the
	// single-use race, so the caller must treat the code as invalid.
	Consume(ctx context.Context, id string) (bool, error)
	// IncrementAttempts atomically increments the attempt counter while it is
	// below maxAttempts, so concurrent wrong guesses cannot exceed the budget.
	IncrementAttempts(ctx context.Context, id string, maxAttempts int) error
	// InvalidateByUser marks every active code of a user/purpose as consumed,
	// voiding outstanding codes before a fresh one is issued.
	InvalidateByUser(ctx context.Context, userID string, purpose Purpose) error
	// InvalidateByUserChannel marks every active code of a user/purpose/channel
	// as consumed. It is used when issuing a fresh OTP so only codes on the
	// same delivery channel are invalidated (e.g. registering email+phone must
	// not invalidate the email code when the phone code is issued).
	InvalidateByUserChannel(ctx context.Context, userID string, purpose Purpose, channel Channel) error
}

// PendingContactChangeRepository persists requested email/phone changes that
// await OTP confirmation before being applied.
type PendingContactChangeRepository interface {
	// Save persists a pending contact change. It returns an error when the
	// write fails.
	Save(ctx context.Context, p *PendingContactChange) error
	// FindPendingByNewValue returns the pending change for a channel whose new
	// value matches. Returns nil, nil when none exists.
	FindPendingByNewValue(ctx context.Context, channel Channel, newValue string) (*PendingContactChange, error)
	// MarkApplied flips a pending change to applied at appliedAt, recording
	// when the change took effect.
	MarkApplied(ctx context.Context, id string, appliedAt time.Time) error
}

// RecoveryCodeRepository persists the hashed, single-use MFA fallback codes
// issued when TOTP is activated.
type RecoveryCodeRepository interface {
	// SaveAll stores the hashed recovery codes of a user, replacing any prior
	// set (called when MFA is activated).
	SaveAll(ctx context.Context, userID string, codeHashes []string) error
	// Consume marks a matching unused code as used and reports whether it did.
	// False means no unused code matched, so the code is invalid or reused.
	Consume(ctx context.Context, userID, codeHash string) (bool, error)
	// DeleteAll removes every recovery code of a user, called when MFA is
	// disabled so no stale fallback codes remain.
	DeleteAll(ctx context.Context, userID string) error
}
