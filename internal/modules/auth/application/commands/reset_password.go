package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
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
	auditor        audit.Auditor
	clock          clock.Clock
	otpMaxAttempts int
}

// NewResetPassword builds the use case.
func NewResetPassword(users domain.UserRepository, codes domain.VerificationCodeRepository, refreshTokens domain.RefreshTokenRepository, hasher hash.PasswordHasher, auditor audit.Auditor, clk clock.Clock, otpMaxAttempts int) *ResetPassword {
	return &ResetPassword{users: users, codes: codes, refreshTokens: refreshTokens, hasher: hasher, auditor: auditor, clock: clk, otpMaxAttempts: otpMaxAttempts}
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
		// Uniform with a wrong code: never reveal whether the identifier is
		// registered.
		return domain.ErrInvalid
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
	if uc.auditor != nil {
		// OldValues intentionally omitted: the previous password hash must not
		// be stored in the audit trail.
		_ = uc.auditor.Record(ctx, audit.Entry{
			SubjectType: "users",
			SubjectID:   user.ID,
			Action:      audit.ActionUpdated,
			NewValues:   map[string]any{"password": true},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return uc.refreshTokens.RevokeByUserID(ctx, user.ID)
}
