package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// ResetPasswordCommand resets the password with the delivered code.
type ResetPasswordCommand struct {
	Identifier  string
	Code        string
	NewPassword string
}

// ResetPassword validates the reset code and replaces the password.
type ResetPassword struct {
	users          domain.UserRepository
	codes          domain.VerificationCodeRepository
	refreshTokens  domain.RefreshTokenRepository
	hasher         hash.PasswordHasher
	clock          clock.Clock
	otpMaxAttempts int
}

// NewResetPassword builds the use case.
func NewResetPassword(users domain.UserRepository, codes domain.VerificationCodeRepository, refreshTokens domain.RefreshTokenRepository, hasher hash.PasswordHasher, clk clock.Clock, otpMaxAttempts int) *ResetPassword {
	return &ResetPassword{users: users, codes: codes, refreshTokens: refreshTokens, hasher: hasher, clock: clk, otpMaxAttempts: otpMaxAttempts}
}

// Execute runs the use case.
func (uc *ResetPassword) Execute(ctx context.Context, cmd ResetPasswordCommand) error {
	identifier := strings.ToLower(strings.TrimSpace(cmd.Identifier))
	if identifier == "" || cmd.Code == "" || len(cmd.NewPassword) < 8 {
		return domain.ErrInvalid
	}
	user, err := findByIdentifier(ctx, uc.users, identifier)
	if err != nil {
		return err
	}
	if user == nil {
		return domain.ErrNotFound
	}

	channel := domain.ChannelPhone
	if strings.Contains(identifier, "@") {
		channel = domain.ChannelEmail
	}
	code, err := uc.codes.FindLatestActive(ctx, user.ID, domain.PurposeReset, channel)
	if err != nil {
		return err
	}
	if err := validateCode(ctx, uc.codes, code, cmd.Code, uc.otpMaxAttempts); err != nil {
		return err
	}

	newHash, err := uc.hasher.Hash(ctx, cmd.NewPassword)
	if err != nil {
		return err
	}
	user.SetPasswordHash(newHash, uc.clock.Now())
	if err := uc.users.Update(ctx, user); err != nil {
		return err
	}
	return uc.refreshTokens.RevokeByUserID(ctx, user.ID)
}
