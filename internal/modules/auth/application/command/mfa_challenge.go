package command

import (
	"context"
	"crypto/rand"
	"math/big"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/application/totp"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

const mfaChallengePrefix = "auth:mfa:"

// NewMFAChallenges builds the challenge store. A nil cache makes every MFA
// login fail closed (challenges cannot be issued).
func NewMFAChallenges(c cache.Cache, ttl time.Duration) *mfaChallenges {
	return &mfaChallenges{cache: c, ttl: ttl}
}

// mfaChallenges issues and redeems single-use, short-lived MFA challenge
// tokens used by the second step of a TOTP-protected login.
type mfaChallenges struct {
	cache cache.Cache
	ttl   time.Duration
}

// issue returns a one-time challenge token bound to userID and the login
// identifier, stored in cache until it is redeemed or the TTL lapses. The
// identifier lets VerifyMFA reset the login rate limiter. Without a cache it
// fails closed.
func (c *mfaChallenges) issue(ctx context.Context, userID, identifier string) (string, error) {
	token := domain.NewOpaqueToken()
	if c.cache == nil {
		// No cache: fail closed, MFA logins cannot start a challenge.
		return "", domain.ErrInvalid
	}
	// userID never contains '|' (UUID v7) and identifiers are emails or
	// E.164 phone numbers, so the separator is unambiguous.
	value := userID + "|" + identifier
	if err := c.cache.Set(ctx, mfaChallengePrefix+token, []byte(value), c.ttl); err != nil {
		return "", err
	}
	return token, nil
}

// take validates a challenge token, consumes it atomically (single-use), and
// returns the bound user id and login identifier. Any missing, expired, or
// repeated token yields ErrInvalid.
func (c *mfaChallenges) take(ctx context.Context, token string) (userID, identifier string, err error) {
	if c.cache == nil || strings.TrimSpace(token) == "" {
		return "", "", domain.ErrInvalid
	}
	// GetDelete removes the entry in one atomic operation, so two concurrent
	// redemptions cannot both succeed. A failed delete (cache error) fails the
	// redemption closed.
	b, err := c.cache.GetDelete(ctx, mfaChallengePrefix+strings.TrimSpace(token))
	if err != nil {
		return "", "", domain.ErrInvalid
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", "", domain.ErrInvalid
	}
	return parts[0], parts[1], nil
}

// recoveryCodeAlphabet avoids easily-confused characters (0,O,1,I).
const recoveryCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// generateRecoveryCodes returns n random one-time codes. They are shown to the
// user once at MFA activation and stored only as SHA-256 hashes.
func generateRecoveryCodes(n int) ([]string, error) {
	if n < 1 {
		n = 8
	}
	codes := make([]string, 0, n)
	for i := 0; i < n; i++ {
		b := make([]byte, 10)
		for j := range b {
			v, err := rand.Int(rand.Reader, big.NewInt(int64(len(recoveryCodeAlphabet))))
			if err != nil {
				return nil, err
			}
			b[j] = recoveryCodeAlphabet[v.Int64()]
		}
		codes = append(codes, string(b))
	}
	return codes, nil
}

// validateMFACode accepts a TOTP code (with one period of clock skew) or a
// single-use recovery code, which is consumed on success.
func validateMFACode(ctx context.Context, user *domain.User, code string, recovery domain.RecoveryCodeRepository) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, nil
	}
	ok, err := totp.Validate(user.TOTPSecret, code, 1)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	return recovery.Consume(ctx, user.ID, domain.HashSecret(code))
}
