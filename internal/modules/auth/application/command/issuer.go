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

// TokenResult carries freshly issued credentials: the access token, the raw
// refresh token, and the access-token lifetime.
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
	// maxSessions caps concurrent session families per user; 0 disables the cap.
	maxSessions int
	// denylist invalidates the access tokens of evicted sessions.
	denylist *jtiDenylist
}

// NewTokenIssuer builds the shared issuer. roles may be nil when RBAC is not
// wired; the access token then carries no roles.
func NewTokenIssuer(tokens token.Manager, refreshTokens domain.RefreshTokenRepository, roles port.Roles, auditor audit.Recorder, accessTTL, refreshTTL time.Duration, clk clock.Clock, maxSessions int, denylist *jtiDenylist) *TokenIssuer {
	return &TokenIssuer{tokens: tokens, refreshTokens: refreshTokens, roles: roles, auditor: auditor, accessTTL: accessTTL, refreshTTL: refreshTTL, clock: clk, maxSessions: maxSessions, denylist: denylist}
}

// Issue mints a fresh credential pair for userID, opening a new session family
// (a brand new login). The oldest families are revoked when the session cap is
// exceeded.
func (i *TokenIssuer) Issue(ctx context.Context, userID string) (*TokenResult, error) {
	return i.IssueInFamily(ctx, userID, "")
}

// IssueInFamily mints a credential pair. An empty familyID opens a new session
// and enforces the per-user session cap; a non-empty one continues an existing
// session (refresh rotation), which never bumps the cap.
func (i *TokenIssuer) IssueInFamily(ctx context.Context, userID, familyID string) (*TokenResult, error) {
	var roles []string
	if i.roles != nil {
		r, _, err := i.roles.RolesAndPermissions(ctx, userID)
		if err != nil {
			return nil, err
		}
		roles = r
	}
	if familyID == "" {
		familyID = domain.NewID()
		if err := i.enforceSessionCap(ctx, userID); err != nil {
			return nil, err
		}
	}
	jti := domain.NewID()
	access, err := i.tokens.IssueAccessToken(ctx, token.Claims{UserID: userID, Roles: roles, JTI: jti}, i.accessTTL)
	if err != nil {
		return nil, err
	}
	raw := domain.NewOpaqueToken()
	rt := domain.NewRefreshToken(userID, raw, familyID, jti, i.refreshTTL, i.clock.Now())
	if err := i.refreshTokens.Save(ctx, rt); err != nil {
		return nil, err
	}
	if i.auditor != nil {
		audit.RecordBestEffort(ctx, i.auditor, audit.Entry{
			SubjectType: "refresh_tokens",
			SubjectID:   rt.ID,
			Action:      audit.ActionCreated,
			NewValues:   map[string]any{"user_id": userID, "family_id": familyID},
			Actor:       audit.ActorFrom(ctx),
		})
	}
	return &TokenResult{AccessToken: access, RefreshToken: raw, ExpiresIn: i.accessTTL}, nil
}

// enforceSessionCap revokes the oldest session families once the user exceeds
// the configured cap, so a user cannot accumulate unbounded sessions. Evicted
// families have their outstanding access tokens denied immediately, matching
// every other revocation path.
func (i *TokenIssuer) enforceSessionCap(ctx context.Context, userID string) error {
	if i.maxSessions <= 0 {
		return nil
	}
	families, err := i.refreshTokens.ListActiveFamilies(ctx, userID, i.clock.Now())
	if err != nil {
		return err
	}
	excess := len(families) - (i.maxSessions - 1)
	for _, f := range families {
		if excess <= 0 {
			break
		}
		if jtis, err := i.refreshTokens.JtisByFamily(ctx, f.ID); err == nil {
			i.denylist.deny(ctx, jtis)
		}
		if err := i.refreshTokens.RevokeFamily(ctx, f.ID); err != nil {
			return err
		}
		excess--
	}
	return nil
}
