package command

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// MagicLinkVerifyCommand exchanges a one-time magic link token for a fresh
// credential pair.
type MagicLinkVerifyCommand struct {
	Token string
}

// MagicLinkVerify consumes a magic-link token and, when valid, issues the
// user's credentials through the shared token issuer. Accounts with MFA
// enabled do not receive tokens from a link: they get a one-time MFA
// challenge that must be completed via VerifyMFA, keeping the second-factor
// guarantee across every login path.
type MagicLinkVerify struct {
	codes      domain.VerificationCodeRepository
	users      domain.UserRepository
	issuer     *TokenIssuer
	challenges *mfaChallenges
}

// NewMagicLinkVerify builds the magic-link exchange use case from the code and
// user repositories, the shared token issuer and the MFA challenge store.
func NewMagicLinkVerify(codes domain.VerificationCodeRepository, users domain.UserRepository, issuer *TokenIssuer, challenges *mfaChallenges) *MagicLinkVerify {
	return &MagicLinkVerify{codes: codes, users: users, issuer: issuer, challenges: challenges}
}

// Execute consumes the magic-link token and issues credentials. It returns
// ErrInvalid for a blank token, ErrNotFound for an unknown or already-consumed
// token, ErrCodeExpired for an expired one, and ErrUnauthorized for suspended
// accounts or a lost single-use race. For MFA-enabled accounts the returned
// result carries only MFAChallenge (no tokens); the caller must complete the
// challenge via VerifyMFA.
func (uc *MagicLinkVerify) Execute(ctx context.Context, cmd MagicLinkVerifyCommand) (*LoginResult, error) {
	token := strings.TrimSpace(cmd.Token)
	if token == "" {
		return nil, domain.ErrInvalid
	}
	code, err := uc.codes.FindActiveByHash(ctx, domain.PurposeMagicLink, domain.HashSecret(token))
	if err != nil {
		return nil, err
	}
	if code == nil {
		return nil, domain.ErrNotFound
	}
	if code.IsConsumed() {
		return nil, domain.ErrNotFound
	}
	if code.IsExpired(nowUTC()) {
		return nil, domain.ErrCodeExpired
	}
	user, err := uc.users.FindByID(ctx, code.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsSuspended() {
		return nil, domain.ErrUnauthorized
	}
	ok, err := uc.codes.Consume(ctx, code.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		// Lost the single-use race: another request already consumed the link.
		return nil, domain.ErrNotFound
	}
	if user.IsTOTPEnabled() {
		// The link is consumed, but no credentials are handed out: the owner
		// must still prove the second factor before a session is created.
		identifier := ""
		if user.Email != nil {
			identifier = *user.Email
		} else if user.Phone != nil {
			identifier = *user.Phone
		}
		challenge, err := uc.challenges.issue(ctx, user.ID, identifier)
		if err != nil {
			return nil, err
		}
		return &LoginResult{MFAChallenge: challenge, User: user}, nil
	}
	res, err := uc.issuer.Issue(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &LoginResult{TokenResult: res, User: user}, nil
}
