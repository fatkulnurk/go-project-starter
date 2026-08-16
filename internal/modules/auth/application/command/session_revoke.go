package command

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// SessionRevokeCommand revokes one session (family) of the user. The caller's
// own session can be revoked too (sign out this device).
type SessionRevokeCommand struct {
	UserID   string
	FamilyID string
}

// SessionRevoke signs out a single session: it denies the session's
// outstanding access tokens and revokes every refresh token in the family.
type SessionRevoke struct {
	refreshTokens domain.RefreshTokenRepository
	auditor       audit.Recorder
	denylist      *jtiDenylist
}

// NewSessionRevoke builds the session-revocation use case from the
// refresh-token repository, the auditor and the shared denylist.
func NewSessionRevoke(refreshTokens domain.RefreshTokenRepository, auditor audit.Recorder, denylist *jtiDenylist) *SessionRevoke {
	return &SessionRevoke{refreshTokens: refreshTokens, auditor: auditor, denylist: denylist}
}

// Execute runs the use case. It only revokes families that belong to the user,
// so a caller cannot sign out sessions it does not own.
func (uc *SessionRevoke) Execute(ctx context.Context, cmd SessionRevokeCommand) error {
	if cmd.UserID == "" || cmd.FamilyID == "" {
		return domain.ErrInvalid
	}
	families, err := uc.refreshTokens.ListActiveFamilies(ctx, cmd.UserID, nowUTC())
	if err != nil {
		return err
	}
	found := false
	for _, f := range families {
		if f.ID == cmd.FamilyID {
			found = true
			break
		}
	}
	if !found {
		return domain.ErrNotFound
	}
	if jtis, err := uc.refreshTokens.JtisByFamily(ctx, cmd.FamilyID); err == nil {
		uc.denylist.deny(ctx, jtis)
	}
	if err := uc.refreshTokens.RevokeFamily(ctx, cmd.FamilyID); err != nil {
		return err
	}
	if uc.auditor != nil {
		audit.RecordBestEffort(ctx, uc.auditor, audit.Entry{
			SubjectType: "sessions",
			SubjectID:   cmd.FamilyID,
			Action:      audit.ActionDeleted,
			OldValues:   map[string]any{"user_id": cmd.UserID},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
