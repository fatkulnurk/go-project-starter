package command

import (
	"context"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// ProcessMagicLinkResult carries the outcome of processing a magic-link
// delivery. User is nil when the email does not match an account; the caller
// must then skip sending.
type ProcessMagicLinkResult struct {
	User *domain.User
	Link string
}

// ProcessMagicLink resolves a magic-link delivery request to an account,
// issues a one-time login token, and returns what the worker needs to send. It
// runs inside the queue worker, so unknown emails are silently skipped instead
// of revealing registration status on the HTTP path.
type ProcessMagicLink struct {
	users    domain.UserRepository
	codes    domain.VerificationCodeRepository
	baseURL  string
	magicTTL time.Duration
	clock    clock.Clock
}

// NewProcessMagicLink builds the use case.
func NewProcessMagicLink(users domain.UserRepository, codes domain.VerificationCodeRepository, baseURL string, magicTTL time.Duration, clk clock.Clock) *ProcessMagicLink {
	return &ProcessMagicLink{users: users, codes: codes, baseURL: baseURL, magicTTL: magicTTL, clock: clk}
}

// Execute resolves email to a user and, when found, issues a magic link. A nil
// User in the result means the email is not registered: skip.
func (uc *ProcessMagicLink) Execute(ctx context.Context, email string) (*ProcessMagicLinkResult, error) {
	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return &ProcessMagicLinkResult{}, nil
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
	return &ProcessMagicLinkResult{User: user, Link: link}, nil
}
