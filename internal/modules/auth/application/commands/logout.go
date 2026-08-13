package commands

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// LogoutCommand revokes a refresh token.
type LogoutCommand struct {
	UserID       string
	RefreshToken string
}

// Logout revokes the presented refresh token.
type Logout struct {
	refreshTokens domain.RefreshTokenRepository
}

// NewLogout builds the use case.
func NewLogout(refreshTokens domain.RefreshTokenRepository) *Logout {
	return &Logout{refreshTokens: refreshTokens}
}

// Execute runs the use case. A missing or already-revoked token is not an
// error.
func (uc *Logout) Execute(ctx context.Context, cmd LogoutCommand) error {
	if cmd.RefreshToken == "" {
		return nil
	}
	t, err := uc.refreshTokens.FindByHash(ctx, domain.HashSecret(cmd.RefreshToken))
	if err != nil {
		return err
	}
	if t == nil || t.UserID != cmd.UserID || t.IsRevoked() {
		return nil
	}
	return uc.refreshTokens.RevokeByID(ctx, t.ID)
}
