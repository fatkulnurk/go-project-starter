package command

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// VerifyEmailCommand verifies an email address with its OTP. OldCode confirms
// the previous address when a contact change is being applied.
type VerifyEmailCommand struct {
	Email   string
	Code    string
	OldCode string
}

// VerifyEmail confirms an email. It applies a pending contact change when the
// address is waiting for re-verification; otherwise it verifies the email
// already stored on the account (registration flow).
type VerifyEmail struct {
	users          domain.UserRepository
	codes          domain.VerificationCodeRepository
	pending        domain.PendingContactChangeRepository
	auditor        audit.Recorder
	clock          clock.Clock
	otpMaxAttempts int
}

// NewVerifyEmail builds the email-verification use case from the user, code
// and pending-change repositories, the auditor, clock and attempt limit.
func NewVerifyEmail(users domain.UserRepository, codes domain.VerificationCodeRepository, pending domain.PendingContactChangeRepository, auditor audit.Recorder, clk clock.Clock, otpMaxAttempts int) *VerifyEmail {
	return &VerifyEmail{users: users, codes: codes, pending: pending, auditor: auditor, clock: clk, otpMaxAttempts: otpMaxAttempts}
}

// Execute verifies an email with its OTP, applying a pending contact change
// when one exists. It is idempotent for already-verified addresses and returns
// ErrInvalid for malformed input or wrong codes, never revealing whether the
// email is registered.
func (uc *VerifyEmail) Execute(ctx context.Context, cmd VerifyEmailCommand) error {
	email, err := domain.NormalizeEmail(cmd.Email)
	if err != nil || cmd.Code == "" {
		return domain.ErrInvalid
	}

	if pending, err := uc.pending.FindPendingByNewValue(ctx, domain.ChannelEmail, email); err != nil {
		return err
	} else if pending != nil {
		return uc.applyPending(ctx, pending, email, cmd.Code, cmd.OldCode)
	}

	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user == nil {
		// Uniform with a wrong code: never reveal whether the email is
		// registered.
		return domain.ErrInvalid
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
	if err := uc.users.Update(ctx, user); err != nil {
		return err
	}
	uc.audit(ctx, "users", user.ID, audit.ActionUpdated,
		map[string]any{"email_verified": false},
		map[string]any{"email_verified": true},
	)
	return nil
}

// applyPending confirms the OTP for a pending contact change and applies the
// new value to the user. When the account had a previous address, a second OTP
// sent to it must also be confirmed so a stolen session cannot hijack the
// account.
func (uc *VerifyEmail) applyPending(ctx context.Context, pending *domain.PendingContactChange, email, code, oldCode string) error {
	user, err := uc.users.FindByID(ctx, pending.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return domain.ErrNotFound
	}
	// Validate the old-channel code first: consuming the new-value code before
	// the old one checks out would burn the new code on a single typo, forcing
	// the user to re-request codes entirely.
	if pending.OldValue != "" {
		old, err := uc.codes.FindLatestActive(ctx, user.ID, domain.PurposeVerifyOld, domain.ChannelEmail)
		if err != nil {
			return err
		}
		if err := validateCode(ctx, uc.codes, old, oldCode, uc.otpMaxAttempts); err != nil {
			return err
		}
	}
	vc, err := uc.codes.FindLatestActive(ctx, user.ID, domain.PurposeVerify, domain.ChannelEmail)
	if err != nil {
		return err
	}
	if err := validateCode(ctx, uc.codes, vc, code, uc.otpMaxAttempts); err != nil {
		return err
	}
	user.SetEmail(email, uc.clock.Now())
	user.VerifyEmail(uc.clock.Now())
	if err := uc.users.Update(ctx, user); err != nil {
		return err
	}
	if err := uc.pending.MarkApplied(ctx, pending.ID, uc.clock.Now()); err != nil {
		return err
	}
	uc.audit(ctx, "pending_contact_changes", pending.ID, audit.ActionUpdated,
		map[string]any{"status": string(domain.PendingStatusPending)},
		map[string]any{"status": string(domain.PendingStatusApplied)},
	)
	uc.audit(ctx, "users", user.ID, audit.ActionUpdated,
		map[string]any{"email": pending.OldValue, "email_verified": false},
		map[string]any{"email": email, "email_verified": true},
	)
	return nil
}

func (uc *VerifyEmail) audit(ctx context.Context, subjectType, subjectID string, action audit.Action, oldValues, newValues map[string]any) {
	if uc.auditor == nil {
		return
	}
	audit.RecordBestEffort(ctx, uc.auditor, audit.Entry{
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Action:      action,
		OldValues:   oldValues,
		NewValues:   newValues,
		Actor:       audit.ActorFrom(ctx),
	})
}
