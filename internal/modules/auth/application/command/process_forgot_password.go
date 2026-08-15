package command

import (
	"context"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/otp"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// ProcessForgotPasswordResult carries the outcome of processing a reset-code
// delivery. User is nil when the identifier does not match an account; the
// caller must then skip sending.
type ProcessForgotPasswordResult struct {
	User *domain.User
	Code string
}

// ProcessForgotPassword resolves a forgot-password delivery request to an
// account, issues a fresh reset code, and returns what the worker needs to
// send. It runs inside the queue worker, so unknown identifiers are silently
// skipped instead of revealing registration status on the HTTP path.
type ProcessForgotPassword struct {
	users     domain.UserRepository
	codes     domain.VerificationCodeRepository
	otpLength int
	otpTTL    time.Duration
	clock     clock.Clock
}

// NewProcessForgotPassword builds the use case.
func NewProcessForgotPassword(users domain.UserRepository, codes domain.VerificationCodeRepository, otpLength int, otpTTL time.Duration, clk clock.Clock) *ProcessForgotPassword {
	return &ProcessForgotPassword{users: users, codes: codes, otpLength: otpLength, otpTTL: otpTTL, clock: clk}
}

// Execute resolves identifier to a user and, when found, issues a reset code.
// A nil User in the result means the identifier is not registered: skip.
func (uc *ProcessForgotPassword) Execute(ctx context.Context, identifier string) (*ProcessForgotPasswordResult, error) {
	user, err := findByIdentifier(ctx, uc.users, identifier)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return &ProcessForgotPasswordResult{}, nil
	}

	channel := domain.ChannelPhone
	if strings.Contains(identifier, "@") {
		channel = domain.ChannelEmail
	}
	if err := uc.codes.InvalidateByUser(ctx, user.ID, domain.PurposeReset); err != nil {
		return nil, err
	}
	code, err := otp.Generate(uc.otpLength)
	if err != nil {
		return nil, err
	}
	vc, err := domain.NewVerificationCode(user.ID, channel, domain.PurposeReset, code, uc.otpTTL, uc.clock.Now())
	if err != nil {
		return nil, err
	}
	if err := uc.codes.Save(ctx, vc); err != nil {
		return nil, err
	}
	return &ProcessForgotPasswordResult{User: user, Code: code}, nil
}
