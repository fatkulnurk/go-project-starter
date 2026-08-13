package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// VerifyPhoneCommand verifies a phone number with its OTP.
type VerifyPhoneCommand struct {
	Phone string
	Code  string
}

// VerifyPhone marks a phone as verified when the OTP matches.
type VerifyPhone struct {
	users          domain.UserRepository
	codes          domain.VerificationCodeRepository
	clock          clock.Clock
	otpMaxAttempts int
}

// NewVerifyPhone builds the use case.
func NewVerifyPhone(users domain.UserRepository, codes domain.VerificationCodeRepository, clk clock.Clock, otpMaxAttempts int) *VerifyPhone {
	return &VerifyPhone{users: users, codes: codes, clock: clk, otpMaxAttempts: otpMaxAttempts}
}

// Execute runs the use case. It is idempotent for already-verified phones.
func (uc *VerifyPhone) Execute(ctx context.Context, cmd VerifyPhoneCommand) error {
	phone := strings.TrimSpace(cmd.Phone)
	if phone == "" || cmd.Code == "" {
		return domain.ErrInvalid
	}
	user, err := uc.users.FindByPhone(ctx, phone)
	if err != nil {
		return err
	}
	if user == nil {
		return domain.ErrNotFound
	}
	if user.IsPhoneVerified() {
		return nil
	}
	code, err := uc.codes.FindLatestActive(ctx, user.ID, domain.PurposeVerify, domain.ChannelPhone)
	if err != nil {
		return err
	}
	if err := validateCode(ctx, uc.codes, code, cmd.Code, uc.otpMaxAttempts); err != nil {
		return err
	}
	user.VerifyPhone(uc.clock.Now())
	return uc.users.Update(ctx, user)
}
