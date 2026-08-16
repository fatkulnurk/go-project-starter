package command

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// DisableTOTPCommand turns off MFA. Code must be a valid TOTP code or a
// one-time recovery code, proving control of the account.
type DisableTOTPCommand struct {
	UserID string
	Code   string
}

// DisableTOTP deactivates MFA after proof of control and removes the stored
// recovery codes.
type DisableTOTP struct {
	users    domain.UserRepository
	recovery domain.RecoveryCodeRepository
	audit    audit.Recorder
	clock    clock.Clock
}

// NewDisableTOTP builds the MFA-deactivation use case from the user and
// recovery-code repositories, the auditor and the clock.
func NewDisableTOTP(users domain.UserRepository, recovery domain.RecoveryCodeRepository, auditor audit.Recorder, clk clock.Clock) *DisableTOTP {
	return &DisableTOTP{users: users, recovery: recovery, audit: auditor, clock: clk}
}

// Execute disables MFA only after the caller proves control of the account
// with a valid TOTP or recovery code. It returns ErrNotFound for unknown users,
// ErrInvalid when MFA is not enabled, and ErrUnauthorized for a bad code. The
// stored recovery codes are deleted once MFA is off.
func (uc *DisableTOTP) Execute(ctx context.Context, cmd DisableTOTPCommand) error {
	user, err := uc.users.FindByID(ctx, cmd.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return domain.ErrNotFound
	}
	if !user.IsTOTPEnabled() {
		return domain.ErrInvalid
	}
	ok, err := validateMFACode(ctx, user, cmd.Code, uc.recovery)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrUnauthorized
	}
	user.DisableTOTP(uc.clock.Now())
	if err := uc.users.Update(ctx, user); err != nil {
		return err
	}
	if err := uc.recovery.DeleteAll(ctx, user.ID); err != nil {
		return err
	}
	if uc.audit != nil {
		audit.RecordBestEffort(ctx, uc.audit, audit.Entry{
			SubjectType: "users",
			SubjectID:   user.ID,
			Action:      audit.ActionUpdated,
			OldValues:   map[string]any{"totp": "enabled"},
			NewValues:   map[string]any{"totp": "disabled"},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
