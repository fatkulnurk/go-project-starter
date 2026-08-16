package command

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// ChangePasswordCommand replaces the password after confirming the current
// one, and signs out every existing session.
type ChangePasswordCommand struct {
	UserID      string
	OldPassword string
	NewPassword string
}

// ChangePassword verifies the current password, sets a new one, and revokes
// all sessions (the caller re-authenticates with the new password).
type ChangePassword struct {
	users         domain.UserRepository
	hasher        hash.PasswordHasher
	refreshTokens domain.RefreshTokenRepository
	auditor       audit.Recorder
	clock         clock.Clock
	denylist      *jtiDenylist
}

// NewChangePassword builds the change-password use case from the user and
// refresh-token repositories, the hasher, auditor, clock and shared denylist.
func NewChangePassword(users domain.UserRepository, hasher hash.PasswordHasher, refreshTokens domain.RefreshTokenRepository, auditor audit.Recorder, clk clock.Clock, denylist *jtiDenylist) *ChangePassword {
	return &ChangePassword{users: users, hasher: hasher, refreshTokens: refreshTokens, auditor: auditor, clock: clk, denylist: denylist}
}

// Execute verifies the current password, stores the new hash, and signs out
// every existing session. It returns ErrInvalid for weak input, ErrNotFound
// when the user is unknown, and ErrUnauthorized when the old password is wrong.
func (uc *ChangePassword) Execute(ctx context.Context, cmd ChangePasswordCommand) error {
	if cmd.OldPassword == "" || len(cmd.NewPassword) < 8 {
		return domain.ErrInvalid
	}
	user, err := uc.users.FindByID(ctx, cmd.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return domain.ErrNotFound
	}
	if !uc.hasher.Compare(ctx, cmd.OldPassword, user.PasswordHash) {
		return domain.ErrUnauthorized
	}
	newHash, err := uc.hasher.Hash(ctx, cmd.NewPassword)
	if err != nil {
		return err
	}
	user.SetPasswordHash(newHash, uc.clock.Now())
	if err := uc.users.Update(ctx, user); err != nil {
		return err
	}
	// A password change must sign out every existing session immediately.
	if jtis, err := uc.refreshTokens.JtisByUser(ctx, user.ID); err == nil {
		uc.denylist.deny(ctx, jtis)
	}
	if err := uc.refreshTokens.RevokeByUserID(ctx, user.ID); err != nil {
		return err
	}
	if uc.auditor != nil {
		// OldValues intentionally omitted: the previous password hash must not
		// be stored in the audit trail.
		audit.RecordBestEffort(ctx, uc.auditor, audit.Entry{
			SubjectType: "users",
			SubjectID:   user.ID,
			Action:      audit.ActionUpdated,
			NewValues:   map[string]any{"password": true},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return nil
}
