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

// UserStatus describes the lifecycle of a user account.
type UserStatus string

// User statuses.
const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
)

// User is an account in the system.
type User struct {
	ID              string
	Name            string
	Email           *string
	Phone           *string
	PasswordHash    string
	EmailVerifiedAt *time.Time
	PhoneVerifiedAt *time.Time
	Status          UserStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewUser builds a user from registration input. At least one of email/phone
// is required, and the password must already be hashed by the caller.
func NewUser(name, email, phone, passwordHash string) (*User, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrInvalid
	}
	if strings.TrimSpace(email) == "" && strings.TrimSpace(phone) == "" {
		return nil, ErrInvalid
	}
	now := time.Now().UTC()
	u := &User{
		ID:           newID(),
		Name:         name,
		PasswordHash: passwordHash,
		Status:       UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if email = strings.TrimSpace(email); email != "" {
		u.Email = &email
	}
	if phone = strings.TrimSpace(phone); phone != "" {
		u.Phone = &phone
	}
	return u, nil
}

// IsEmailVerified reports whether the email address was verified.
func (u *User) IsEmailVerified() bool { return u.EmailVerifiedAt != nil }

// IsPhoneVerified reports whether the phone number was verified.
func (u *User) IsPhoneVerified() bool { return u.PhoneVerifiedAt != nil }

// VerifyEmail marks the email as verified.
func (u *User) VerifyEmail(now time.Time) {
	t := now.UTC()
	u.EmailVerifiedAt = &t
	u.UpdatedAt = now.UTC()
}

// VerifyPhone marks the phone as verified.
func (u *User) VerifyPhone(now time.Time) {
	t := now.UTC()
	u.PhoneVerifiedAt = &t
	u.UpdatedAt = now.UTC()
}

// SetPasswordHash updates the stored password hash.
func (u *User) SetPasswordHash(hash string, now time.Time) {
	u.PasswordHash = hash
	u.UpdatedAt = now.UTC()
}

// IsSuspended reports whether the account is locked.
func (u *User) IsSuspended() bool { return u.Status == UserStatusSuspended }
