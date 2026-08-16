package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// RefreshCommand rotates a refresh token into a fresh credential pair. The
// refresh token is the raw value issued at login or at the last rotation.
type RefreshCommand struct {
	RefreshToken string
}

// Refresh rotates a valid refresh token within its session family, revoking
// the presented token as part of the exchange.
type Refresh struct {
	refreshTokens domain.RefreshTokenRepository
	users         domain.UserRepository
	issuer        *TokenIssuer
	denylist      *jtiDenylist
}

// NewRefresh builds the refresh use case from the token and user repositories,
// the shared issuer and the denylist.
func NewRefresh(refreshTokens domain.RefreshTokenRepository, users domain.UserRepository, issuer *TokenIssuer, denylist *jtiDenylist) *Refresh {
	return &Refresh{refreshTokens: refreshTokens, users: users, issuer: issuer, denylist: denylist}
}

// Execute rotates a refresh token into a fresh credential pair within the same
// session family. It returns ErrUnauthorized for unknown, expired, or reused
// tokens; a reused (already rotated) token revokes the whole family as a theft
// signal. The rotation itself is atomic against concurrent requests.
func (uc *Refresh) Execute(ctx context.Context, cmd RefreshCommand) (*TokenResult, error) {
	raw := strings.TrimSpace(cmd.RefreshToken)
	if raw == "" {
		return nil, domain.ErrUnauthorized
	}
	t, err := uc.refreshTokens.FindByHash(ctx, domain.HashSecret(raw))
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, domain.ErrUnauthorized
	}
	if t.IsRevoked() {
		// Reuse of an already-rotated credential means the token was stolen:
		// revoke the whole session family and its outstanding access tokens.
		uc.revokeFamily(ctx, t.FamilyID)
		return nil, domain.ErrUnauthorized
	}
	if t.IsExpired(nowUTC()) {
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
		uc.revokeFamily(ctx, t.FamilyID)
		return nil, domain.ErrUnauthorized
	}
	// Continue the same session family so the session count is unchanged.
	return uc.issuer.IssueInFamily(ctx, user.ID, t.FamilyID)
}

// revokeFamily revokes every refresh token of a session family and denies its
// outstanding access tokens. Revocation errors are swallowed: the operation is
// best-effort and can be retried.
func (uc *Refresh) revokeFamily(ctx context.Context, familyID string) {
	if familyID == "" {
		return
	}
	if jtis, err := uc.refreshTokens.JtisByFamily(ctx, familyID); err == nil {
		uc.denylist.deny(ctx, jtis)
	}
	if err := uc.refreshTokens.RevokeFamily(ctx, familyID); err != nil {
		// The family stays partially revoked on retry; nothing else to do.
		return
	}
}
