package command

import (
	"context"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// LoginCommand is the input for password login. Identifier may be an email or
// a phone number. Code carries the optional MFA second factor (TOTP or a
// one-time recovery code) when the account has MFA enabled.
type LoginCommand struct {
	Identifier string
	Password   string
	Code       string
	IP         string
}

// LoginResult wraps the issued credentials together with the signed-in user.
// MFAChallenge is non-empty when the account requires a second factor and the
// caller must complete it via VerifyMFA before receiving credentials.
type LoginResult struct {
	*TokenResult
	User         *domain.User
	MFAChallenge string
}

// Login authenticates by identifier and password and issues tokens, inserting
// an MFA challenge step when the account has TOTP enabled.
type Login struct {
	users                domain.UserRepository
	hasher               hash.PasswordHasher
	issuer               *TokenIssuer
	requireEmailVerified bool
	rateLimiter          *rateLimiter
	recovery             domain.RecoveryCodeRepository
	challenges           *mfaChallenges
	countryCode          string
	// dummyHash is compared against when the identifier does not exist so the
	// request burns the same bcrypt time as a wrong password, preventing a
	// timing side-channel that would reveal which identifiers are registered.
	dummyHash string
}

// NewLogin builds the login use case from the user, hasher and issuer ports,
// the shared rate limiter, and the recovery and MFA challenge stores.
func NewLogin(users domain.UserRepository, hasher hash.PasswordHasher, issuer *TokenIssuer, requireEmailVerified bool, rateLimiter *rateLimiter, recovery domain.RecoveryCodeRepository, challenges *mfaChallenges, countryCode string) *Login {
	dummy, _ := hasher.Hash(context.Background(), "dummy-password-for-timing-equalization")
	return &Login{users: users, hasher: hasher, issuer: issuer, requireEmailVerified: requireEmailVerified, rateLimiter: rateLimiter, recovery: recovery, challenges: challenges, countryCode: countryCode, dummyHash: dummy}
}

// Execute authenticates by identifier and password and, when the account has
// MFA, returns a challenge for the second factor. It returns ErrUnauthorized
// for wrong credentials, suspended accounts, or unverified emails (uniform
// with "unknown identifier" to prevent enumeration), ErrTooManyAttempts when
// the rate limit is exceeded, and a token pair otherwise.
func (uc *Login) Execute(ctx context.Context, cmd LoginCommand) (*LoginResult, error) {
	identifier := strings.ToLower(strings.TrimSpace(cmd.Identifier))
	if identifier == "" || cmd.Password == "" {
		return nil, domain.ErrInvalid
	}
	if err := uc.rateLimiter.Check(ctx, identifier, cmd.IP); err != nil {
		return nil, err
	}

	user, err := findByIdentifier(ctx, uc.users, identifier, uc.countryCode)
	if err != nil {
		return nil, err
	}
	if user == nil {
		// Burn the same hashing time as a real comparison so the response time
		// does not reveal whether the identifier is registered.
		uc.hasher.Compare(ctx, cmd.Password, uc.dummyHash)
		return nil, domain.ErrUnauthorized
	}
	if !uc.hasher.Compare(ctx, cmd.Password, user.PasswordHash) {
		return nil, domain.ErrUnauthorized
	}
	if user.IsSuspended() {
		return nil, domain.ErrUnauthorized
	}
	// Unverified accounts respond exactly like a wrong password: a distinct
	// "verification required" error would let attackers enumerate which
	// identifiers are registered and their verification state.
	if uc.requireEmailVerified && user.Email != nil && !user.IsEmailVerified() {
		return nil, domain.ErrUnauthorized
	}

	if user.IsTOTPEnabled() {
		if strings.TrimSpace(cmd.Code) == "" {
			// Step one done: hand back a short-lived challenge the client
			// exchanges for the code in VerifyMFA.
			challenge, err := uc.challenges.issue(ctx, user.ID, identifier)
			if err != nil {
				return nil, err
			}
			return &LoginResult{MFAChallenge: challenge, User: user}, nil
		}
		ok, err := validateMFACode(ctx, user, cmd.Code, uc.recovery)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, domain.ErrUnauthorized
		}
	}

	res, err := uc.issuer.Issue(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	uc.rateLimiter.Reset(ctx, identifier, cmd.IP)
	return &LoginResult{TokenResult: res, User: user}, nil
}

// findByIdentifier resolves an email or phone to a user. Returns nil,nil when
// no user matches (the caller must treat it as unauthorized). Phones are
// normalized so "0812..." and "+62812..." resolve to the same account.
func findByIdentifier(ctx context.Context, users domain.UserRepository, identifier, countryCode string) (*domain.User, error) {
	if strings.Contains(identifier, "@") {
		return users.FindByEmail(ctx, identifier)
	}
	phone, err := domain.NormalizePhone(identifier, countryCode)
	if err != nil {
		return nil, nil
	}
	return users.FindByPhone(ctx, phone)
}

// rateLimiter counts requests per key in cache. It backs login attempt
// limiting as well as per-target OTP/magic-link request throttling so a single
// address cannot be mail-bombed.
type rateLimiter struct {
	cache  cache.Cache
	max    int64
	window time.Duration
	prefix string
}

// NewRateLimiter builds a rate limiter that counts requests under prefix.
// Distinct prefixes keep the login, forgot-password and magic-link flows on
// independent counters.
func NewRateLimiter(c cache.Cache, max int64, window time.Duration, prefix string) *rateLimiter {
	return &rateLimiter{cache: c, max: max, window: window, prefix: prefix}
}

func (l *rateLimiter) key(target, ip string) string {
	return l.prefix + ":" + domain.HashSecret(target) + ":" + ip
}

// Check increments the counter for the (target, ip) key and returns
// ErrTooManyAttempts when the count exceeds the configured maximum. The window
// is re-applied on every increment (sliding window); setting it only on the
// first request lets a concurrent DB-driver increment clobber the expiration.
func (l *rateLimiter) Check(ctx context.Context, target, ip string) error {
	key := l.key(target, ip)
	n, err := l.cache.Increment(ctx, key, 1)
	if err != nil {
		return err
	}
	_ = l.cache.Expire(ctx, key, l.window)
	if n > l.max {
		return domain.ErrTooManyAttempts
	}
	return nil
}

// Reset clears the counter for the (target, ip) key after a successful action,
// so a correct attempt does not keep counting toward the limit.
func (l *rateLimiter) Reset(ctx context.Context, target, ip string) {
	_ = l.cache.Delete(ctx, l.key(target, ip))
}
