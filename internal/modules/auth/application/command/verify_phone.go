package command

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// VerifyPhoneCommand verifies a phone number with its OTP. OldCode confirms
// the previous number when a contact change is being applied.
type VerifyPhoneCommand struct {
	Phone   string
	Code    string
	OldCode string
}

// VerifyPhone confirms a phone. It applies a pending contact change when the
// number is waiting for re-verification; otherwise it verifies the phone
// already stored on the account (registration flow).
type VerifyPhone struct {
	users          domain.UserRepository
	codes          domain.VerificationCodeRepository
	pending        domain.PendingContactChangeRepository
	auditor        audit.Recorder
	clock          clock.Clock
	otpMaxAttempts int
	// countryCode expands local phone numbers (leading 0) into E.164.
	countryCode string
}

// NewVerifyPhone builds the phone-verification use case from the user, code
// and pending-change repositories, the auditor, clock and attempt limit.
func NewVerifyPhone(users domain.UserRepository, codes domain.VerificationCodeRepository, pending domain.PendingContactChangeRepository, auditor audit.Recorder, clk clock.Clock, otpMaxAttempts int, countryCode string) *VerifyPhone {
	return &VerifyPhone{users: users, codes: codes, pending: pending, auditor: auditor, clock: clk, otpMaxAttempts: otpMaxAttempts, countryCode: countryCode}
}

// Execute verifies a phone with its OTP, applying a pending contact change
// when one exists. It is idempotent for already-verified numbers and returns
// ErrInvalid for malformed input or wrong codes, never revealing whether the
// phone is registered.
func (uc *VerifyPhone) Execute(ctx context.Context, cmd VerifyPhoneCommand) error {
	phone, err := domain.NormalizePhone(cmd.Phone, uc.countryCode)
	if err != nil || cmd.Code == "" {
		return domain.ErrInvalid
	}

	if pending, err := uc.pending.FindPendingByNewValue(ctx, domain.ChannelPhone, phone); err != nil {
		return err
	} else if pending != nil {
		return uc.applyPending(ctx, pending, phone, cmd.Code, cmd.OldCode)
	}

	user, err := uc.users.FindByPhone(ctx, phone)
	if err != nil {
		return err
	}
	if user == nil {
		// Uniform with a wrong code: never reveal whether the phone is
		// registered.
		return domain.ErrInvalid
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
	if err := uc.users.Update(ctx, user); err != nil {
		return err
	}
	uc.audit(ctx, "users", user.ID, audit.ActionUpdated,
		map[string]any{"phone_verified": false},
		map[string]any{"phone_verified": true},
	)
	return nil
}

// applyPending confirms the OTP for a pending contact change and applies the
// new value to the user. When the account had a previous number, a second OTP
// sent to it must also be confirmed so a stolen session cannot hijack the
// account.
func (uc *VerifyPhone) applyPending(ctx context.Context, pending *domain.PendingContactChange, phone, code, oldCode string) error {
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
		old, err := uc.codes.FindLatestActive(ctx, user.ID, domain.PurposeVerifyOld, domain.ChannelPhone)
		if err != nil {
			return err
		}
		if err := validateCode(ctx, uc.codes, old, oldCode, uc.otpMaxAttempts); err != nil {
			return err
		}
	}
	vc, err := uc.codes.FindLatestActive(ctx, user.ID, domain.PurposeVerify, domain.ChannelPhone)
	if err != nil {
		return err
	}
	if err := validateCode(ctx, uc.codes, vc, code, uc.otpMaxAttempts); err != nil {
		return err
	}
	user.SetPhone(phone, uc.clock.Now())
	user.VerifyPhone(uc.clock.Now())
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
		map[string]any{"phone": pending.OldValue, "phone_verified": false},
		map[string]any{"phone": phone, "phone_verified": true},
	)
	return nil
}

func (uc *VerifyPhone) audit(ctx context.Context, subjectType, subjectID string, action audit.Action, oldValues, newValues map[string]any) {
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
