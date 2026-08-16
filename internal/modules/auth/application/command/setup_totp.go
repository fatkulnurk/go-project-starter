package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/application/totp"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// SetupTOTPCommand starts MFA activation. The returned secret is shown once so
// the user can enroll it in an authenticator app; it only takes effect after a
// valid code is confirmed via ConfirmTOTP.
type SetupTOTPCommand struct {
	UserID string
}

// SetupTOTPResult carries the generated secret and the provisioning URI used
// to enroll it in an authenticator app.
type SetupTOTPResult struct {
	Secret string
	URI    string
}

// SetupTOTP generates and stages a new TOTP secret for the user without
// activating it; activation happens in ConfirmTOTP.
type SetupTOTP struct {
	users  domain.UserRepository
	audit  audit.Recorder
	clock  clock.Clock
	issuer string
}

// NewSetupTOTP builds the use case. issuer is the app name shown by
// authenticator apps.
func NewSetupTOTP(users domain.UserRepository, auditor audit.Recorder, clk clock.Clock, issuer string) *SetupTOTP {
	return &SetupTOTP{users: users, audit: auditor, clock: clk, issuer: issuer}
}

// Execute generates and stages a fresh TOTP secret for the user, returning it
// together with the provisioning URI. It returns ErrNotFound for unknown users
// and ErrConflict when MFA is already enabled.
func (uc *SetupTOTP) Execute(ctx context.Context, cmd SetupTOTPCommand) (*SetupTOTPResult, error) {
	user, err := uc.users.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrNotFound
	}
	if user.IsTOTPEnabled() {
		return nil, domain.ErrConflict
	}
	secret, err := totp.GenerateSecret()
	if err != nil {
		return nil, err
	}
	user.StageTOTP(secret, uc.clock.Now())
	if err := uc.users.Update(ctx, user); err != nil {
		return nil, err
	}
	account := user.Name
	if user.Email != nil {
		account = *user.Email
	} else if user.Phone != nil {
		account = *user.Phone
	}
	if uc.audit != nil {
		audit.RecordBestEffort(ctx, uc.audit, audit.Entry{
			SubjectType: "users",
			SubjectID:   user.ID,
			Action:      audit.ActionUpdated,
			NewValues:   map[string]any{"totp": "staged"},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return &SetupTOTPResult{
		Secret: secret,
		URI:    totp.ProvisioningURI(strings.TrimSpace(uc.issuer), account, secret),
	}, nil
}
