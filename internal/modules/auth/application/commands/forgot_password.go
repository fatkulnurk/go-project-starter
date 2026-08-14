package commands

import (
	"context"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/otp"
	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/tasks"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// ForgotPasswordCommand requests a password-reset code for an email or phone.
type ForgotPasswordCommand struct {
	Identifier string
	IP         string
}

// ForgotPasswordResult reports the outcome; the code is only returned in dev
// mode.
type ForgotPasswordResult struct {
	DevCode   string
	ExpiresIn time.Duration
}

// ForgotPassword issues a reset code and delivers it on the matching channel.
type ForgotPassword struct {
	users     domain.UserRepository
	codes     domain.VerificationCodeRepository
	enqueuer  queue.Enqueuer
	clock     clock.Clock
	otpLength int
	otpTTL    time.Duration
	devMode   bool
	limiter   *rateLimiter
}

// NewForgotPassword builds the use case.
func NewForgotPassword(users domain.UserRepository, codes domain.VerificationCodeRepository, enqueuer queue.Enqueuer, clk clock.Clock, otpLength int, otpTTL time.Duration, devMode bool, limiter *rateLimiter) *ForgotPassword {
	return &ForgotPassword{users: users, codes: codes, enqueuer: enqueuer, clock: clk, otpLength: otpLength, otpTTL: otpTTL, devMode: devMode, limiter: limiter}
}

// Execute runs the use case.
func (uc *ForgotPassword) Execute(ctx context.Context, cmd ForgotPasswordCommand) (*ForgotPasswordResult, error) {
	identifier := strings.ToLower(strings.TrimSpace(cmd.Identifier))
	if identifier == "" {
		return nil, domain.ErrInvalid
	}
	if uc.limiter != nil {
		if err := uc.limiter.Check(ctx, identifier, cmd.IP); err != nil {
			return nil, err
		}
	}
	user, err := findByIdentifier(ctx, uc.users, identifier)
	if err != nil {
		return nil, err
	}
	if user == nil {
		// No-op: return the same success shape as a real send so the response
		// does not reveal whether the account exists. No code is issued and no
		// email/SMS is enqueued.
		return &ForgotPasswordResult{ExpiresIn: uc.otpTTL}, nil
	}

	channel := domain.ChannelPhone
	isEmail := strings.Contains(identifier, "@")
	if isEmail {
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

	if isEmail {
		err = tasks.EnqueueForgotPasswordEmail(ctx, uc.enqueuer, tasks.ForgotPasswordEmailPayload{
			To: *user.Email, Name: user.Name, Code: code,
		})
	} else {
		err = tasks.EnqueueForgotPasswordSMS(ctx, uc.enqueuer, tasks.ForgotPasswordSMSPayload{
			To: *user.Phone, Code: code,
		})
	}
	if err != nil {
		return nil, err
	}

	res := &ForgotPasswordResult{ExpiresIn: uc.otpTTL}
	if uc.devMode {
		res.DevCode = code
	}
	return res, nil
}
