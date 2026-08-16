package command

import (
	"context"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/task"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// MagicLinkRequestCommand requests a magic login link for an email address.
// IP is used only for rate limiting.
type MagicLinkRequestCommand struct {
	Email string
	IP    string
}

// MagicLinkRequestResult reports the outcome. It is identical whether or not
// the account exists, so the response never reveals registration status. The
// actual account lookup and link issuance happen in the worker.
type MagicLinkRequestResult struct {
	ExpiresIn time.Duration
}

// MagicLinkRequest always enqueues a magic-link delivery task and returns a
// uniform success. The worker resolves the email to a user and skips the send
// when the account does not exist.
type MagicLinkRequest struct {
	enqueuer queue.Enqueuer
	magicTTL time.Duration
	limiter  *rateLimiter
}

// NewMagicLinkRequest builds the magic-link request use case from the enqueuer,
// the link TTL and the rate limiter.
func NewMagicLinkRequest(enqueuer queue.Enqueuer, magicTTL time.Duration, limiter *rateLimiter) *MagicLinkRequest {
	return &MagicLinkRequest{enqueuer: enqueuer, magicTTL: magicTTL, limiter: limiter}
}

// Execute normalizes the email, rate-checks the request, and enqueues a
// magic-link delivery task. It returns a uniform success (ErrInvalid only for
// malformed email) so registration status is never revealed.
func (uc *MagicLinkRequest) Execute(ctx context.Context, cmd MagicLinkRequestCommand) (*MagicLinkRequestResult, error) {
	email, err := domain.NormalizeEmail(cmd.Email)
	if err != nil {
		return nil, domain.ErrInvalid
	}
	if uc.limiter != nil {
		if err := uc.limiter.Check(ctx, email, cmd.IP); err != nil {
			return nil, err
		}
	}
	if err := task.EnqueueMagicLinkEmail(ctx, uc.enqueuer, task.MagicLinkEmailPayload{
		Email: email,
	}); err != nil {
		return nil, err
	}
	return &MagicLinkRequestResult{ExpiresIn: uc.magicTTL}, nil
}
