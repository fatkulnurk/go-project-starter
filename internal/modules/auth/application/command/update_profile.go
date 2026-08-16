package command

import (
	"context"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// UpdateProfileCommand updates the mutable profile fields. Empty values keep
// the current field; a changed email or phone records a pending change and
// issues a new OTP that must be confirmed via VerifyEmail/VerifyPhone before
// the new value replaces the old one.
type UpdateProfileCommand struct {
	UserID string
	Name   string
	Email  string
	Phone  string
}

// UpdateProfileResult reports the saved user plus dev-mode codes for any
// channel that just issued a re-verification OTP.
type UpdateProfileResult struct {
	User         *domain.User
	DevEmailCode string
	DevPhoneCode string
}

// UpdateProfile updates a user's profile. It is authenticated: the caller's
// user ID comes from the token, never from the request body.
type UpdateProfile struct {
	users     domain.UserRepository
	codes     domain.VerificationCodeRepository
	pending   domain.PendingContactChangeRepository
	enqueuer  queue.Enqueuer
	auditor   audit.Recorder
	clock     clock.Clock
	otpLength int
	otpTTL    time.Duration
	devMode   bool
	// countryCode expands local phone numbers (leading 0) into E.164.
	countryCode string
}

// NewUpdateProfile builds the profile-update use case from the user, code and
// pending-change repositories plus the enqueuer, auditor and clock.
func NewUpdateProfile(users domain.UserRepository, codes domain.VerificationCodeRepository, pending domain.PendingContactChangeRepository, enqueuer queue.Enqueuer, auditor audit.Recorder, clk clock.Clock, otpLength int, otpTTL time.Duration, devMode bool, countryCode string) *UpdateProfile {
	return &UpdateProfile{users: users, codes: codes, pending: pending, enqueuer: enqueuer, auditor: auditor, clock: clk, otpLength: otpLength, otpTTL: otpTTL, devMode: devMode, countryCode: countryCode}
}

// Execute updates the mutable profile fields, recording a pending contact
// change and issuing re-verification OTPs when the email or phone changes. It
// returns ErrNotFound for unknown users and ErrConflict when the new email or
// phone is already taken, including by another user's pending change.
func (uc *UpdateProfile) Execute(ctx context.Context, cmd UpdateProfileCommand) (*UpdateProfileResult, error) {
	user, err := uc.users.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrNotFound
	}

	res := &UpdateProfileResult{}
	now := uc.clock.Now()
	oldName := user.Name

	if name := strings.TrimSpace(cmd.Name); name != "" {
		user.SetName(name, now)
	}

	if rawEmail := strings.TrimSpace(cmd.Email); rawEmail != "" {
		email, err := domain.NormalizeEmail(rawEmail)
		if err != nil {
			return nil, err
		}
		// Compare canonical forms: resubmitting the current address with
		// different casing/format is a no-op instead of a pending change.
		if user.Email != nil && *user.Email == email {
			// no-op
		} else {
			if existing, err := uc.users.FindByEmail(ctx, email); err != nil {
				return nil, err
			} else if existing != nil && existing.ID != user.ID {
				return nil, domain.ErrConflict
			}
			if pending, err := uc.pending.FindPendingByNewValue(ctx, domain.ChannelEmail, email); err != nil {
				return nil, err
			} else if pending != nil && pending.UserID != user.ID {
				return nil, domain.ErrConflict
			}
			oldValue := ""
			if user.Email != nil {
				oldValue = *user.Email
			}
			if err := uc.savePending(ctx, user.ID, domain.ChannelEmail, oldValue, email); err != nil {
				return nil, err
			}
			code, err := issueVerificationCode(ctx, uc.codes, uc.enqueuer, user.ID, user.Name, domain.ChannelEmail, email, uc.otpLength, uc.otpTTL, uc.clock)
			if err != nil {
				return nil, err
			}
			if uc.devMode {
				res.DevEmailCode = code
			}
			if oldValue != "" {
				// Confirm the change on the OLD channel too, so a stolen session
				// cannot silently move the account to an attacker's address.
				if _, err := issueCode(ctx, uc.codes, uc.enqueuer, user.ID, user.Name, domain.ChannelEmail, domain.PurposeVerifyOld, oldValue, uc.otpLength, uc.otpTTL, uc.clock); err != nil {
					return nil, err
				}
			}
			uc.audit(ctx, user.ID, "pending_contact_changes", audit.ActionCreated, nil, map[string]any{"channel": string(domain.ChannelEmail), "old_value": oldValue, "new_value": email})
		}
	}

	if rawPhone := strings.TrimSpace(cmd.Phone); rawPhone != "" {
		phone, err := domain.NormalizePhone(rawPhone, uc.countryCode)
		if err != nil {
			return nil, err
		}
		// Compare canonical forms so resubmitting the current number in a
		// different format is a no-op instead of a pending change.
		if user.Phone != nil && *user.Phone == phone {
			// no-op
		} else {
			if existing, err := uc.users.FindByPhone(ctx, phone); err != nil {
				return nil, err
			} else if existing != nil && existing.ID != user.ID {
				return nil, domain.ErrConflict
			}
			if pending, err := uc.pending.FindPendingByNewValue(ctx, domain.ChannelPhone, phone); err != nil {
				return nil, err
			} else if pending != nil && pending.UserID != user.ID {
				return nil, domain.ErrConflict
			}
			oldValue := ""
			if user.Phone != nil {
				oldValue = *user.Phone
			}
			if err := uc.savePending(ctx, user.ID, domain.ChannelPhone, oldValue, phone); err != nil {
				return nil, err
			}
			code, err := issueVerificationCode(ctx, uc.codes, uc.enqueuer, user.ID, user.Name, domain.ChannelPhone, phone, uc.otpLength, uc.otpTTL, uc.clock)
			if err != nil {
				return nil, err
			}
			if uc.devMode {
				res.DevPhoneCode = code
			}
			if oldValue != "" {
				if _, err := issueCode(ctx, uc.codes, uc.enqueuer, user.ID, user.Name, domain.ChannelPhone, domain.PurposeVerifyOld, oldValue, uc.otpLength, uc.otpTTL, uc.clock); err != nil {
					return nil, err
				}
			}
			uc.audit(ctx, user.ID, "pending_contact_changes", audit.ActionCreated, nil, map[string]any{"channel": string(domain.ChannelPhone), "old_value": oldValue, "new_value": phone})
		}
	}

	if err := uc.users.Update(ctx, user); err != nil {
		return nil, err
	}
	if user.Name != oldName {
		uc.audit(ctx, "users", user.ID, audit.ActionUpdated,
			map[string]any{"name": oldName},
			map[string]any{"name": user.Name},
		)
	}
	res.User = user
	return res, nil
}

func (uc *UpdateProfile) savePending(ctx context.Context, userID string, channel domain.Channel, oldValue, newValue string) error {
	return uc.pending.Save(ctx, domain.NewPendingContactChange(userID, channel, oldValue, newValue, uc.clock.Now()))
}

func (uc *UpdateProfile) audit(ctx context.Context, subjectID string, subjectType string, action audit.Action, oldValues, newValues map[string]any) {
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
