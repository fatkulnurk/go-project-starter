package command

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/application/totp"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// ConfirmTOTPCommand activates MFA after verifying the staged secret with a
// code from the authenticator app.
type ConfirmTOTPCommand struct {
	UserID string
	Code   string
}

// ConfirmTOTPResult carries the one-time recovery codes issued on activation.
// They are shown exactly once and stored only as hashes.
type ConfirmTOTPResult struct {
	RecoveryCodes []string
}

// ConfirmTOTP activates a staged TOTP secret after a valid code and issues
// one-time recovery codes for fallback access.
type ConfirmTOTP struct {
	users    domain.UserRepository
	recovery domain.RecoveryCodeRepository
	audit    audit.Recorder
	clock    clock.Clock
}

// NewConfirmTOTP builds the MFA-activation use case from the user and
// recovery-code repositories, the auditor and the clock.
func NewConfirmTOTP(users domain.UserRepository, recovery domain.RecoveryCodeRepository, auditor audit.Recorder, clk clock.Clock) *ConfirmTOTP {
	return &ConfirmTOTP{users: users, recovery: recovery, audit: auditor, clock: clk}
}

// Execute validates the code against the staged TOTP secret and, on success,
// activates MFA and issues one-time recovery codes. It returns ErrNotFound for
// unknown users, ErrInvalid when nothing is staged or the code is wrong, and
// ErrConflict when MFA is already enabled.
func (uc *ConfirmTOTP) Execute(ctx context.Context, cmd ConfirmTOTPCommand) (*ConfirmTOTPResult, error) {
	user, err := uc.users.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrNotFound
	}
	if user.TOTPSecret == "" {
		return nil, domain.ErrInvalid // nothing staged
	}
	if user.IsTOTPEnabled() {
		return nil, domain.ErrConflict
	}
	ok, err := totp.Validate(user.TOTPSecret, cmd.Code, 1)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrInvalid
	}
	// Persist the recovery-code set BEFORE flipping MFA on: if generation or
	// storage fails, MFA stays off and the client can retry activation instead
	// of being locked into an enforced MFA with no codes. A later retry replaces
	// the stored hashes, so a partial write is harmless.
	codes, err := generateRecoveryCodes(8)
	if err != nil {
		return nil, err
	}
	hashes := make([]string, len(codes))
	for i, c := range codes {
		hashes[i] = domain.HashSecret(c)
	}
	if err := uc.recovery.SaveAll(ctx, user.ID, hashes); err != nil {
		return nil, err
	}
	user.ConfirmTOTP(uc.clock.Now())
	if err := uc.users.Update(ctx, user); err != nil {
		return nil, err
	}
	if uc.audit != nil {
		audit.RecordBestEffort(ctx, uc.audit, audit.Entry{
			SubjectType: "users",
			SubjectID:   user.ID,
			Action:      audit.ActionUpdated,
			NewValues:   map[string]any{"totp": "enabled"},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return &ConfirmTOTPResult{RecoveryCodes: codes}, nil
}
