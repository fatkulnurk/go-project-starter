package command

import (
	"context"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/task"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// ForgotPasswordCommand requests a password-reset code for an email or phone.
type ForgotPasswordCommand struct {
	Identifier string
	IP         string
}

// ForgotPasswordResult reports the outcome. It is identical whether or not the
// account exists, so the response never reveals registration status. The actual
// account lookup and code issuance happen in the worker.
type ForgotPasswordResult struct {
	ExpiresIn time.Duration
}

// ForgotPassword always enqueues a reset-code delivery task and returns a
// uniform success. The worker resolves the identifier to a user and skips the
// send when the account does not exist.
type ForgotPassword struct {
	enqueuer queue.Enqueuer
	otpTTL   time.Duration
	limiter  *rateLimiter
}

// NewForgotPassword builds the use case.
func NewForgotPassword(enqueuer queue.Enqueuer, otpTTL time.Duration, limiter *rateLimiter) *ForgotPassword {
	return &ForgotPassword{enqueuer: enqueuer, otpTTL: otpTTL, limiter: limiter}
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

	if strings.Contains(identifier, "@") {
		err := task.EnqueueForgotPasswordEmail(ctx, uc.enqueuer, task.ForgotPasswordEmailPayload{
			Identifier: identifier,
		})
		if err != nil {
			return nil, err
		}
	} else {
		err := task.EnqueueForgotPasswordSMS(ctx, uc.enqueuer, task.ForgotPasswordSMSPayload{
			Identifier: identifier,
		})
		if err != nil {
			return nil, err
		}
	}

	return &ForgotPasswordResult{ExpiresIn: uc.otpTTL}, nil
}
