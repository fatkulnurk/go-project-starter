package command

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
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
	auditor       audit.Recorder
}

// NewLogout builds the use case.
func NewLogout(refreshTokens domain.RefreshTokenRepository, auditor audit.Recorder) *Logout {
	return &Logout{refreshTokens: refreshTokens, auditor: auditor}
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
	if err := uc.refreshTokens.RevokeByID(ctx, t.ID); err != nil {
		return err
	}
	if uc.auditor != nil {
		_ = uc.auditor.Record(ctx, audit.Entry{
			SubjectType: "refresh_tokens",
			SubjectID:   t.ID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"user_id": t.UserID},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
