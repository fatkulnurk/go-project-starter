package commands

import (
	"context"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/token"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/ports"
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
	roles         ports.Roles
	accessTTL     time.Duration
	refreshTTL    time.Duration
	clock         clock.Clock
}

// NewTokenIssuer builds the shared issuer. roles may be nil when RBAC is not
// wired; the access token then carries no roles.
func NewTokenIssuer(tokens token.Manager, refreshTokens domain.RefreshTokenRepository, roles ports.Roles, accessTTL, refreshTTL time.Duration, clk clock.Clock) *TokenIssuer {
	return &TokenIssuer{tokens: tokens, refreshTokens: refreshTokens, roles: roles, accessTTL: accessTTL, refreshTTL: refreshTTL, clock: clk}
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
	return &TokenResult{AccessToken: access, RefreshToken: raw, ExpiresIn: i.accessTTL}, nil
}
