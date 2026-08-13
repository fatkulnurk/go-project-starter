package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// VerifyEmailCommand verifies an email address with its OTP.
type VerifyEmailCommand struct {
	Email string
	Code  string
}

// VerifyEmail marks an email as verified when the OTP matches.
type VerifyEmail struct {
	users          domain.UserRepository
	codes          domain.VerificationCodeRepository
	clock          clock.Clock
	otpMaxAttempts int
}

// NewVerifyEmail builds the use case.
func NewVerifyEmail(users domain.UserRepository, codes domain.VerificationCodeRepository, clk clock.Clock, otpMaxAttempts int) *VerifyEmail {
	return &VerifyEmail{users: users, codes: codes, clock: clk, otpMaxAttempts: otpMaxAttempts}
}

// Execute runs the use case. It is idempotent for already-verified emails.
func (uc *VerifyEmail) Execute(ctx context.Context, cmd VerifyEmailCommand) error {
	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	if email == "" || cmd.Code == "" {
		return domain.ErrInvalid
	}
	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user == nil {
		return domain.ErrNotFound
	}
	if user.IsEmailVerified() {
		return nil
	}
	code, err := uc.codes.FindLatestActive(ctx, user.ID, domain.PurposeVerify, domain.ChannelEmail)
	if err != nil {
		return err
	}
	if err := validateCode(ctx, uc.codes, code, cmd.Code, uc.otpMaxAttempts); err != nil {
		return err
	}
	user.VerifyEmail(uc.clock.Now())
	return uc.users.Update(ctx, user)
}
