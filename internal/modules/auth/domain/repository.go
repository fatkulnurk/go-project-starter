package domain

import (
	"context"
	"time"
)

// UserRepository persists users.
type UserRepository interface {
	Save(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByPhone(ctx context.Context, phone string) (*User, error)
	Update(ctx context.Context, u *User) error
}

// RefreshTokenRepository persists refresh tokens.
type RefreshTokenRepository interface {
	Save(ctx context.Context, t *RefreshToken) error
	FindByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeByID(ctx context.Context, id string) error
	// RevokeByIDIfActive revokes the token only if it is still active (not yet
	// revoked) and reports whether it did. It is used for atomic rotation so a
	// token can only be exchanged once even under concurrent requests.
	RevokeByIDIfActive(ctx context.Context, id string) (bool, error)
	RevokeByUserID(ctx context.Context, userID string) error
}

// VerificationCodeRepository persists verification codes and magic links.
type VerificationCodeRepository interface {
	Save(ctx context.Context, c *VerificationCode) error
	// FindLatestActive returns the newest non-consumed, non-expired code for a
	// user/purpose/channel. Returns nil, nil when none exists.
	FindLatestActive(ctx context.Context, userID string, purpose Purpose, channel Channel) (*VerificationCode, error)
	// FindActiveByHash returns a non-consumed, non-expired code matching the
	// purpose and secret hash. Returns nil, nil when none exists.
	FindActiveByHash(ctx context.Context, purpose Purpose, codeHash string) (*VerificationCode, error)
	Consume(ctx context.Context, id string) error
	IncrementAttempts(ctx context.Context, id string, attempts int) error
	// InvalidateByUser marks every active code of a user/purpose as consumed.
	InvalidateByUser(ctx context.Context, userID string, purpose Purpose) error
}

// PendingContactChangeRepository persists pending email/phone changes.
type PendingContactChangeRepository interface {
	Save(ctx context.Context, p *PendingContactChange) error
	// FindPendingByNewValue returns the pending change for a channel whose new
	// value matches. Returns nil, nil when none exists.
	FindPendingByNewValue(ctx context.Context, channel Channel, newValue string) (*PendingContactChange, error)
	// MarkApplied flips a pending change to applied.
	MarkApplied(ctx context.Context, id string, appliedAt time.Time) error
}
