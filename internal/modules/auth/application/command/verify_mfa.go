package command

import (
	"context"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// VerifyMFACommand completes a TOTP-protected login with the second factor.
// Challenge comes from the login response when the account requires MFA.
type VerifyMFACommand struct {
	Challenge string
	Code      string
	IP        string
}

// VerifyMFA validates the MFA code for a pending login challenge and issues
// the credential pair. On success it resets the login rate limiter for the
// identifier recorded at challenge time, so legitimate two-step logins do not
// accumulate toward the 429 budget.
type VerifyMFA struct {
	challenges *mfaChallenges
	users      domain.UserRepository
	recovery   domain.RecoveryCodeRepository
	issuer     *TokenIssuer
	limiter    *rateLimiter
}

// NewVerifyMFA builds the MFA second-factor use case from the challenge store,
// the user and recovery-code repositories, the shared token issuer, and the
// login rate limiter to reset on success.
func NewVerifyMFA(challenges *mfaChallenges, users domain.UserRepository, recovery domain.RecoveryCodeRepository, issuer *TokenIssuer, limiter *rateLimiter) *VerifyMFA {
	return &VerifyMFA{challenges: challenges, users: users, recovery: recovery, issuer: issuer, limiter: limiter}
}

// Execute redeems the login challenge with a valid TOTP or recovery code and
// issues the credential pair. It returns ErrInvalid for a bad challenge and
// ErrUnauthorized for a wrong code, a suspended account, or MFA not enabled.
func (uc *VerifyMFA) Execute(ctx context.Context, cmd VerifyMFACommand) (*LoginResult, error) {
	userID, identifier, err := uc.challenges.take(ctx, cmd.Challenge)
	if err != nil {
		return nil, err
	}
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsTOTPEnabled() || user.IsSuspended() {
		return nil, domain.ErrUnauthorized
	}
	ok, err := validateMFACode(ctx, user, cmd.Code, uc.recovery)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	res, err := uc.issuer.Issue(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if uc.limiter != nil && identifier != "" {
		uc.limiter.Reset(ctx, identifier, cmd.IP)
	}
	return &LoginResult{TokenResult: res, User: user}, nil
}
