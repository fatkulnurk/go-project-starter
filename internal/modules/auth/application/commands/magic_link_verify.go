package commands

import (
	"context"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// MagicLinkVerifyCommand exchanges a magic link token for credentials.
type MagicLinkVerifyCommand struct {
	Token string
}

// MagicLinkVerify consumes a magic link and issues tokens.
type MagicLinkVerify struct {
	codes  domain.VerificationCodeRepository
	users  domain.UserRepository
	issuer *TokenIssuer
}

// NewMagicLinkVerify builds the use case.
func NewMagicLinkVerify(codes domain.VerificationCodeRepository, users domain.UserRepository, issuer *TokenIssuer) *MagicLinkVerify {
	return &MagicLinkVerify{codes: codes, users: users, issuer: issuer}
}

// Execute runs the use case.
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
	if err := uc.codes.Consume(ctx, code.ID); err != nil {
		return nil, err
	}
	res, err := uc.issuer.Issue(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &LoginResult{TokenResult: res, User: user}, nil
}
