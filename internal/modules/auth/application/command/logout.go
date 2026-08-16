package command

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// LogoutCommand revokes a refresh token, identified by its raw value. UserID
// scopes the revocation so one user cannot sign out another's session.
type LogoutCommand struct {
	UserID       string
	RefreshToken string
}

// Logout revokes the presented refresh token's whole session family and
// denies its outstanding access tokens.
type Logout struct {
	refreshTokens domain.RefreshTokenRepository
	auditor       audit.Recorder
	denylist      *jtiDenylist
}

// NewLogout builds the logout use case from the refresh-token repository, the
// auditor and the shared denylist.
func NewLogout(refreshTokens domain.RefreshTokenRepository, auditor audit.Recorder, denylist *jtiDenylist) *Logout {
	return &Logout{refreshTokens: refreshTokens, auditor: auditor, denylist: denylist}
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
	if t == nil || t.UserID != cmd.UserID {
		return nil
	}
	if jtis, err := uc.refreshTokens.JtisByFamily(ctx, t.FamilyID); err == nil {
		uc.denylist.deny(ctx, jtis)
	}
	if err := uc.refreshTokens.RevokeFamily(ctx, t.FamilyID); err != nil {
		return err
	}
	if uc.auditor != nil {
		audit.RecordBestEffort(ctx, uc.auditor, audit.Entry{
			SubjectType: "refresh_tokens",
			SubjectID:   t.ID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"user_id": t.UserID},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
