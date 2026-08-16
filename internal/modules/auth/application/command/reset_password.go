package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// ResetPasswordCommand resets the password with the delivered code for the
// given email or phone identifier.
type ResetPasswordCommand struct {
	Identifier  string
	Code        string
	NewPassword string
}

// ResetPassword validates the reset code and replaces the password, signing
// out every existing session in the process.
type ResetPassword struct {
	users          domain.UserRepository
	codes          domain.VerificationCodeRepository
	refreshTokens  domain.RefreshTokenRepository
	hasher         hash.PasswordHasher
	auditor        audit.Recorder
	clock          clock.Clock
	otpMaxAttempts int
	countryCode    string
	denylist       *jtiDenylist
}

// NewResetPassword builds the reset-password use case from the user, code and
// refresh-token repositories, the hasher, auditor, clock and denylist.
func NewResetPassword(users domain.UserRepository, codes domain.VerificationCodeRepository, refreshTokens domain.RefreshTokenRepository, hasher hash.PasswordHasher, auditor audit.Recorder, clk clock.Clock, otpMaxAttempts int, countryCode string, denylist *jtiDenylist) *ResetPassword {
	return &ResetPassword{users: users, codes: codes, refreshTokens: refreshTokens, hasher: hasher, auditor: auditor, clock: clk, otpMaxAttempts: otpMaxAttempts, countryCode: countryCode, denylist: denylist}
}

// Execute validates the reset code and replaces the password, then signs out
// every existing session. It returns ErrInvalid for malformed input, unknown
// identifiers (uniform with a wrong code), or a bad/consumed code, ErrCodeExpired
// for expired codes, and ErrTooManyAttempts when the attempt budget is spent.
func (uc *ResetPassword) Execute(ctx context.Context, cmd ResetPasswordCommand) error {
	identifier, err := normalizeIdentifier(cmd.Identifier, uc.countryCode)
	if err != nil || cmd.Code == "" || len(cmd.NewPassword) < 8 {
		return domain.ErrInvalid
	}
	user, err := findByIdentifier(ctx, uc.users, identifier, uc.countryCode)
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
		audit.RecordBestEffort(ctx, uc.auditor, audit.Entry{
			SubjectType: "users",
			SubjectID:   user.ID,
			Action:      audit.ActionUpdated,
			NewValues:   map[string]any{"password": true},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	// A password reset must sign out every existing session immediately: deny
	// all outstanding access tokens and revoke every refresh token.
	if jtis, err := uc.refreshTokens.JtisByUser(ctx, user.ID); err == nil {
		uc.denylist.deny(ctx, jtis)
	}
	return uc.refreshTokens.RevokeByUserID(ctx, user.ID)
}
