package commands

import (
	"context"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// LoginCommand is the input for password login. Identifier may be an email or
// a phone number.
type LoginCommand struct {
	Identifier string
	Password   string
	IP         string
}

// LoginResult wraps the issued credentials together with the signed-in user.
type LoginResult struct {
	*TokenResult
	User *domain.User
}

// Login authenticates by identifier + password and issues tokens.
type Login struct {
	users                domain.UserRepository
	hasher               hash.PasswordHasher
	issuer               *TokenIssuer
	requireEmailVerified bool
	rateLimiter          *loginRateLimiter
	// dummyHash is compared against when the identifier does not exist so the
	// request burns the same bcrypt time as a wrong password, preventing a
	// timing side-channel that would reveal which identifiers are registered.
	dummyHash string
}

// NewLogin builds the login use case.
func NewLogin(users domain.UserRepository, hasher hash.PasswordHasher, issuer *TokenIssuer, requireEmailVerified bool, rateLimiter *loginRateLimiter) *Login {
	dummy, _ := hasher.Hash(context.Background(), "dummy-password-for-timing-equalization")
	return &Login{users: users, hasher: hasher, issuer: issuer, requireEmailVerified: requireEmailVerified, rateLimiter: rateLimiter, dummyHash: dummy}
}

// Execute runs the use case.
func (uc *Login) Execute(ctx context.Context, cmd LoginCommand) (*LoginResult, error) {
	identifier := strings.ToLower(strings.TrimSpace(cmd.Identifier))
	if identifier == "" || cmd.Password == "" {
		return nil, domain.ErrInvalid
	}
	if err := uc.rateLimiter.Check(ctx, identifier, cmd.IP); err != nil {
		return nil, err
	}

	user, err := findByIdentifier(ctx, uc.users, identifier)
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
	if uc.requireEmailVerified && user.Email != nil && !user.IsEmailVerified() {
		return nil, domain.ErrVerificationRequired
	}

	res, err := uc.issuer.Issue(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	uc.rateLimiter.Reset(ctx, identifier, cmd.IP)
	return &LoginResult{TokenResult: res, User: user}, nil
}

// findByIdentifier resolves an email or phone to a user. Returns nil,nil when
// no user matches (the caller must treat it as unauthorized).
func findByIdentifier(ctx context.Context, users domain.UserRepository, identifier string) (*domain.User, error) {
	if strings.Contains(identifier, "@") {
		return users.FindByEmail(ctx, identifier)
	}
	return users.FindByPhone(ctx, identifier)
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

// NewRateLimiter builds a rate limiter that counts under prefix.
func NewRateLimiter(c cache.Cache, max int64, window time.Duration, prefix string) *rateLimiter {
	return &rateLimiter{cache: c, max: max, window: window, prefix: prefix}
}

func (l *rateLimiter) key(target, ip string) string {
	return l.prefix + ":" + domain.HashSecret(target) + ":" + ip
}

// Check increments the counter; returns ErrTooManyAttempts when over the limit.
func (l *rateLimiter) Check(ctx context.Context, target, ip string) error {
	key := l.key(target, ip)
	n, err := l.cache.Increment(ctx, key, 1)
	if err != nil {
		return err
	}
	if n == 1 {
		_ = l.cache.Expire(ctx, key, l.window)
	}
	if n > l.max {
		return domain.ErrTooManyAttempts
	}
	return nil
}

// Reset clears the counter after a successful action.
func (l *rateLimiter) Reset(ctx context.Context, target, ip string) {
	_ = l.cache.Delete(ctx, l.key(target, ip))
}

// loginRateLimiter is kept for compatibility; its keys are namespaced under
// the login prefix.
type loginRateLimiter struct{ *rateLimiter }

// NewLoginRateLimiter builds a login-scoped rate limiter.
func NewLoginRateLimiter(c cache.Cache, max int64, window time.Duration) *loginRateLimiter {
	return &loginRateLimiter{rateLimiter: NewRateLimiter(c, max, window, "rl:login")}
}

func (l *loginRateLimiter) Check(ctx context.Context, identifier, ip string) error {
	return l.rateLimiter.Check(ctx, identifier, ip)
}

func (l *loginRateLimiter) Reset(ctx context.Context, identifier, ip string) {
	l.rateLimiter.Reset(ctx, identifier, ip)
}
