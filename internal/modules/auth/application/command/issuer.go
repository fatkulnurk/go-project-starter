package command

import (
	"context"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/application/token"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/port"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// TokenResult carries freshly issued credentials.
type TokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}

// TokenIssuer issues access + refresh token pairs. Shared by every login-like
// use case (password login, magic link exchange, refresh).
type TokenIssuer struct {
	tokens        token.Manager
	refreshTokens domain.RefreshTokenRepository
	roles         port.Roles
	auditor       audit.Recorder
	accessTTL     time.Duration
	refreshTTL    time.Duration
	clock         clock.Clock
}

// NewTokenIssuer builds the shared issuer. roles may be nil when RBAC is not
// wired; the access token then carries no roles.
func NewTokenIssuer(tokens token.Manager, refreshTokens domain.RefreshTokenRepository, roles port.Roles, auditor audit.Recorder, accessTTL, refreshTTL time.Duration, clk clock.Clock) *TokenIssuer {
	return &TokenIssuer{tokens: tokens, refreshTokens: refreshTokens, roles: roles, auditor: auditor, accessTTL: accessTTL, refreshTTL: refreshTTL, clock: clk}
}

// Issue mints a fresh credential pair for userID and stores the refresh token.
func (i *TokenIssuer) Issue(ctx context.Context, userID string) (*TokenResult, error) {
	var roles []string
	if i.roles != nil {
		r, _, err := i.roles.RolesAndPermissions(ctx, userID)
		if err != nil {
			return nil, err
		}
		roles = r
	}
	access, err := i.tokens.IssueAccessToken(ctx, token.Claims{UserID: userID, Roles: roles}, i.accessTTL)
	if err != nil {
		return nil, err
	}
	raw := domain.NewOpaqueToken()
	rt := domain.NewRefreshToken(userID, raw, i.refreshTTL, i.clock.Now())
	if err := i.refreshTokens.Save(ctx, rt); err != nil {
		return nil, err
	}
	if i.auditor != nil {
		_ = i.auditor.Record(ctx, audit.Entry{
			SubjectType: "refresh_tokens",
			SubjectID:   rt.ID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"user_id": userID},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return &TokenResult{AccessToken: access, RefreshToken: raw, ExpiresIn: i.accessTTL}, nil
}
