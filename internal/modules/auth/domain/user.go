package domain

import (
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/apierr"
)

// Errors returned by domain constructors and use cases. Aliased to the shared
// API sentinels so the HTTP layer maps them to one consistent response shape.
var (
	ErrNotFound             = apierr.ErrNotFound
	ErrConflict             = apierr.ErrConflict
	ErrInvalid              = apierr.ErrInvalid
	ErrUnauthorized         = apierr.ErrUnauthorized
	ErrVerificationRequired = apierr.ErrVerificationNeeded
	ErrCodeExpired          = apierr.ErrCodeExpired
	ErrTooManyAttempts      = apierr.ErrTooManyRequests
)

// UserStatus describes the lifecycle of a user account: active or suspended.
// Suspended accounts cannot authenticate or refresh their sessions.
type UserStatus string

// User statuses a User can take over its lifetime: active or suspended. A
// suspended account is rejected by the authentication flows.
const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
)

// User is an account in the system with contact addresses, authentication
// state (password, MFA) and a lifecycle status.
type User struct {
	ID              string
	Name            string
	Email           *string
	Phone           *string
	PasswordHash    string
	EmailVerifiedAt *time.Time
	PhoneVerifiedAt *time.Time
	// TOTPSecret is the base32 MFA shared secret; TOTPConfirmedAt non-nil marks
	// MFA as enforced at login. A staged-but-unconfirmed secret is ignored.
	TOTPSecret      string
	TOTPConfirmedAt *time.Time
	Status          UserStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewUser builds a user from registration input. At least one of email/phone
// is required, and the password must already be hashed by the caller. Email
// and phone must pass format validation; use NormalizeEmail/NormalizePhone in
// the caller to canonicalize values before they reach this constructor.
func NewUser(name, email, phone, passwordHash string, now time.Time) (*User, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrInvalid
	}
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)
	if email == "" && phone == "" {
		return nil, ErrInvalid
	}
	if email != "" && !IsValidEmail(email) {
		return nil, ErrInvalid
	}
	if phone != "" && !IsValidPhone(phone) {
		return nil, ErrInvalid
	}
	u := &User{
		ID:           newID(),
		Name:         name,
		PasswordHash: passwordHash,
		Status:       UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if email != "" {
		u.Email = &email
	}
	if phone != "" {
		u.Phone = &phone
	}
	return u, nil
}

// IsEmailVerified reports whether the email address was verified, i.e. the
// EmailVerifiedAt timestamp is set.
func (u *User) IsEmailVerified() bool { return u.EmailVerifiedAt != nil }

// IsPhoneVerified reports whether the phone number was verified, i.e. the
// PhoneVerifiedAt timestamp is set.
func (u *User) IsPhoneVerified() bool { return u.PhoneVerifiedAt != nil }

// VerifyEmail marks the email as verified at now and stamps the record as
// updated.
func (u *User) VerifyEmail(now time.Time) {
	t := now.UTC()
	u.EmailVerifiedAt = &t
	u.UpdatedAt = now.UTC()
}

// VerifyPhone marks the phone as verified at now and stamps the record as
// updated.
func (u *User) VerifyPhone(now time.Time) {
	t := now.UTC()
	u.PhoneVerifiedAt = &t
	u.UpdatedAt = now.UTC()
}

// SetPasswordHash replaces the stored password hash at now. The caller must
// hash the new password before calling.
func (u *User) SetPasswordHash(hash string, now time.Time) {
	u.PasswordHash = hash
	u.UpdatedAt = now.UTC()
}

// SetName updates the display name, trimming surrounding whitespace, and
// stamps the record as updated at now.
func (u *User) SetName(name string, now time.Time) {
	u.Name = strings.TrimSpace(name)
	u.UpdatedAt = now.UTC()
}

// SetEmail replaces the email address and clears its verification so the new
// address must be verified again before it counts as verified.
func (u *User) SetEmail(email string, now time.Time) {
	v := strings.ToLower(strings.TrimSpace(email))
	u.Email = &v
	u.EmailVerifiedAt = nil
	u.UpdatedAt = now.UTC()
}

// SetPhone replaces the phone number and clears its verification so the new
// number must be verified again before it counts as verified.
func (u *User) SetPhone(phone string, now time.Time) {
	v := strings.TrimSpace(phone)
	u.Phone = &v
	u.PhoneVerifiedAt = nil
	u.UpdatedAt = now.UTC()
}

// IsSuspended reports whether the account is locked. Suspended accounts are
// rejected by the authentication and refresh flows.
func (u *User) IsSuspended() bool { return u.Status == UserStatusSuspended }

// IsTOTPEnabled reports whether MFA is enforced for this account, i.e. the
// TOTP secret was confirmed. A staged-only secret does not count.
func (u *User) IsTOTPEnabled() bool { return u.TOTPConfirmedAt != nil }

// StageTOTP stores a freshly generated secret without activating it. MFA only
// takes effect once ConfirmTOTP is called with a valid code.
func (u *User) StageTOTP(secret string, now time.Time) {
	u.TOTPSecret = secret
	u.TOTPConfirmedAt = nil
	u.UpdatedAt = now.UTC()
}

// ConfirmTOTP activates a previously staged TOTP secret at now, turning MFA on
// for the account.
func (u *User) ConfirmTOTP(now time.Time) {
	t := now.UTC()
	u.TOTPConfirmedAt = &t
	u.UpdatedAt = t
}

// DisableTOTP clears the TOTP secret and confirmation timestamp at now, so MFA
// is no longer enforced at login.
func (u *User) DisableTOTP(now time.Time) {
	u.TOTPSecret = ""
	u.TOTPConfirmedAt = nil
	u.UpdatedAt = now.UTC()
}
