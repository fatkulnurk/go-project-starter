package domain

import "context"

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
