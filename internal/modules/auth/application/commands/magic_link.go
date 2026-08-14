package commands

import (
	"context"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/tasks"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// MagicLinkRequestCommand requests a magic login link for an email address.
type MagicLinkRequestCommand struct {
	Email string
	IP    string
}

// MagicLinkRequestResult reports the outcome; the link is only returned in
// dev mode.
type MagicLinkRequestResult struct {
	DevLink   string
	ExpiresIn time.Duration
}

// MagicLinkRequest issues a one-time magic login link and emails it.
type MagicLinkRequest struct {
	users    domain.UserRepository
	codes    domain.VerificationCodeRepository
	enqueuer queue.Enqueuer
	clock    clock.Clock
	baseURL  string
	magicTTL time.Duration
	devMode  bool
	limiter  *rateLimiter
}

// NewMagicLinkRequest builds the use case.
func NewMagicLinkRequest(users domain.UserRepository, codes domain.VerificationCodeRepository, enqueuer queue.Enqueuer, clk clock.Clock, baseURL string, magicTTL time.Duration, devMode bool, limiter *rateLimiter) *MagicLinkRequest {
	return &MagicLinkRequest{users: users, codes: codes, enqueuer: enqueuer, clock: clk, baseURL: baseURL, magicTTL: magicTTL, devMode: devMode, limiter: limiter}
}

// Execute runs the use case.
func (uc *MagicLinkRequest) Execute(ctx context.Context, cmd MagicLinkRequestCommand) (*MagicLinkRequestResult, error) {
	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	if email == "" {
		return nil, domain.ErrInvalid
	}
	if uc.limiter != nil {
		if err := uc.limiter.Check(ctx, email, cmd.IP); err != nil {
			return nil, err
		}
	}
	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		// No-op: return the same success shape as a real send so the response
		// does not reveal whether the account exists. No code is issued and no
		// email is enqueued.
		return &MagicLinkRequestResult{ExpiresIn: uc.magicTTL}, nil
	}

	raw := domain.NewOpaqueToken()
	if err := uc.codes.InvalidateByUser(ctx, user.ID, domain.PurposeMagicLink); err != nil {
		return nil, err
	}
	vc, err := domain.NewVerificationCode(user.ID, domain.ChannelEmail, domain.PurposeMagicLink, raw, uc.magicTTL, uc.clock.Now())
	if err != nil {
		return nil, err
	}
	if err := uc.codes.Save(ctx, vc); err != nil {
		return nil, err
	}

	link := strings.TrimSuffix(uc.baseURL, "/") + "/api/v1/auth/magic-link/verify?token=" + raw
	if err := tasks.EnqueueMagicLinkEmail(ctx, uc.enqueuer, tasks.MagicLinkEmailPayload{
		To: *user.Email, Name: user.Name, Link: link,
	}); err != nil {
		return nil, err
	}

	res := &MagicLinkRequestResult{ExpiresIn: uc.magicTTL}
	if uc.devMode {
		res.DevLink = link
	}
	return res, nil
}
