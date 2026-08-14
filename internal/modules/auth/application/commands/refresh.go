package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// RefreshCommand rotates a refresh token into a fresh credential pair.
type RefreshCommand struct {
	RefreshToken string
}

// Refresh rotates a valid refresh token.
type Refresh struct {
	refreshTokens domain.RefreshTokenRepository
	users         domain.UserRepository
	issuer        *TokenIssuer
}

// NewRefresh builds the use case.
func NewRefresh(refreshTokens domain.RefreshTokenRepository, users domain.UserRepository, issuer *TokenIssuer) *Refresh {
	return &Refresh{refreshTokens: refreshTokens, users: users, issuer: issuer}
}

// Execute runs the use case.
func (uc *Refresh) Execute(ctx context.Context, cmd RefreshCommand) (*TokenResult, error) {
	raw := strings.TrimSpace(cmd.RefreshToken)
	if raw == "" {
		return nil, domain.ErrUnauthorized
	}
	t, err := uc.refreshTokens.FindByHash(ctx, domain.HashSecret(raw))
	if err != nil {
		return nil, err
	}
	if t == nil || t.IsRevoked() || t.IsExpired(nowUTC()) {
		return nil, domain.ErrUnauthorized
	}
	user, err := uc.users.FindByID(ctx, t.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsSuspended() {
		return nil, domain.ErrUnauthorized
	}
	// Atomically revoke the old token. Only one concurrent request can win the
	// rotation; a lost race (token already revoked) is treated as reuse of an
	// already-rotated credential and rejected.
	active, err := uc.refreshTokens.RevokeByIDIfActive(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, domain.ErrUnauthorized
	}
	return uc.issuer.Issue(ctx, user.ID)
}
