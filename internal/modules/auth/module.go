// Package auth is the authentication module: registration, login (password
// and magic link), email/phone verification, forgot/reset password and token
// management. It depends only on cross-cutting application contracts and the
// repositories wired through Dependencies.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	applicationauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
	appaqueue "github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/application/sms"
	"github.com/fatkulnurk/go-project-starter/internal/application/token"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/adapter/api"
	queueadapter "github.com/fatkulnurk/go-project-starter/internal/modules/auth/adapter/queue"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/command"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/query"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/infrastructure"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
	"github.com/go-chi/chi/v5"
)

// mfaChallengeTTL bounds the lifetime of the MFA challenge issued during the
// second step of a TOTP-protected login.
const mfaChallengeTTL = 5 * time.Minute

// Settings carries auth behavior configured in the composition root.
// All fields are set by the application wiring; the module consumes them as-is.
type Settings struct {
	AccessTokenTTL        time.Duration
	RefreshTokenTTL       time.Duration
	OTPLength             int
	OTPTTL                time.Duration
	OTPMaxAttempts        int
	MagicLinkTTL          time.Duration
	RequireEmailVerified  bool
	RateLimitMax          int64
	RateLimitWindow       time.Duration
	PublicRateLimitMax    int64
	PublicRateLimitWindow time.Duration
	BaseURL               string
	AppName               string
	// AssetsBaseURL is the absolute base URL of static assets used in email
	// HTML (defaults to BaseURL when empty).
	AssetsBaseURL string
	// DefaultCountryCode expands local phone numbers (leading 0) into E.164
	// during normalization, e.g. "62" for Indonesia.
	DefaultCountryCode string
	// MaxActiveSessions caps how many concurrent refresh-token families
	// (sessions) a user may hold; the oldest are revoked when exceeded.
	MaxActiveSessions int
	DevMode           bool
}

// Dependencies are the ports the module needs; all wired by the composition
// root.
type Dependencies struct {
	ReadDB   *sql.DB // optional read replica; nil = use DB for reads
	DB       *sql.DB // write database (primary)
	DBDriver string
	Cache    cache.Cache
	Enqueuer appaqueue.Enqueuer
	Mailer   mailer.MailSender
	SMS      sms.Sender
	Tokens   token.Manager
	Hasher   hash.PasswordHasher
	Clock    clock.Clock
	RBAC     rbac.Service
	Auditor  audit.Recorder
	Settings Settings
	// Location is the app timezone for SQL-written timestamps.
	Location *time.Location
}

// Module wires the auth use cases and their adapters. It exposes the use
// cases through API and the adapters through RegisterAPI/RegisterQueue.
type Module struct {
	API      API
	authn    applicationauth.Authenticator
	mailer   mailer.MailSender
	sms      sms.Sender
	settings Settings
	clock    clock.Clock
	cache    cache.Cache
	// processForgot and processMagic run in the worker: they resolve delivery
	// requests to accounts and issue the reset code / magic link.
	processForgot *command.ProcessForgotPassword
	processMagic  *command.ProcessMagicLink
}

// New constructs the auth module, wiring the repositories, use cases and the
// token/MFA rate-limit stores. The returned Module exposes its use cases via
// API and registers them on routers through RegisterAPI and RegisterQueue.
func New(deps Dependencies) *Module {
	// Resolve the read pool: use the dedicated read replica when configured,
	// otherwise fall back to the primary write pool.
	readDB := deps.DB
	if deps.ReadDB != nil {
		readDB = deps.ReadDB
	}

	users := infrastructure.NewUserRepository(readDB, deps.DB, deps.DBDriver, deps.Location)
	refreshTokens := infrastructure.NewRefreshTokenRepository(readDB, deps.DB, deps.DBDriver, deps.Location)
	codes := infrastructure.NewVerificationCodeRepository(readDB, deps.DB, deps.DBDriver, deps.Location)
	pending := infrastructure.NewPendingContactChangeRepository(readDB, deps.DB, deps.DBDriver, deps.Location)
	recovery := infrastructure.NewRecoveryCodeRepository(readDB, deps.DB, deps.DBDriver, deps.Location)

	roles := rbacAdapter{svc: deps.RBAC}

	denylist := command.NewJTIDenylist(deps.Cache, deps.Settings.AccessTokenTTL)
	challenges := command.NewMFAChallenges(deps.Cache, mfaChallengeTTL)
	issuer := command.NewTokenIssuer(deps.Tokens, refreshTokens, roles, deps.Auditor, deps.Settings.AccessTokenTTL, deps.Settings.RefreshTokenTTL, deps.Clock, deps.Settings.MaxActiveSessions, denylist)
	rateLimiter := command.NewRateLimiter(deps.Cache, deps.Settings.RateLimitMax, deps.Settings.RateLimitWindow, "rl:login")
	forgotLimiter := command.NewRateLimiter(deps.Cache, deps.Settings.RateLimitMax, deps.Settings.RateLimitWindow, "rl:forgot")
	magicLimiter := command.NewRateLimiter(deps.Cache, deps.Settings.RateLimitMax, deps.Settings.RateLimitWindow, "rl:magic")

	processForgot := command.NewProcessForgotPassword(users, codes, deps.Settings.OTPLength, deps.Settings.OTPTTL, deps.Clock, deps.Settings.DefaultCountryCode)
	processMagic := command.NewProcessMagicLink(users, codes, deps.Settings.BaseURL, deps.Settings.MagicLinkTTL, deps.Clock)

	return &Module{
		API: API{
			Register:         command.NewRegister(users, codes, deps.Hasher, deps.Enqueuer, roles, deps.Auditor, deps.Clock, deps.Settings.OTPLength, deps.Settings.OTPTTL, deps.Settings.OTPMaxAttempts, deps.Settings.DevMode, deps.Settings.DefaultCountryCode),
			Login:            command.NewLogin(users, deps.Hasher, issuer, deps.Settings.RequireEmailVerified, rateLimiter, recovery, challenges, deps.Settings.DefaultCountryCode),
			MagicLinkRequest: command.NewMagicLinkRequest(deps.Enqueuer, deps.Settings.MagicLinkTTL, magicLimiter),
			MagicLinkVerify:  command.NewMagicLinkVerify(codes, users, issuer, challenges),
			VerifyEmail:      command.NewVerifyEmail(users, codes, pending, deps.Auditor, deps.Clock, deps.Settings.OTPMaxAttempts),
			VerifyPhone:      command.NewVerifyPhone(users, codes, pending, deps.Auditor, deps.Clock, deps.Settings.OTPMaxAttempts, deps.Settings.DefaultCountryCode),
			ForgotPassword:   command.NewForgotPassword(deps.Enqueuer, deps.Settings.OTPTTL, forgotLimiter, deps.Settings.DefaultCountryCode),
			ResetPassword:    command.NewResetPassword(users, codes, refreshTokens, deps.Hasher, deps.Auditor, deps.Clock, deps.Settings.OTPMaxAttempts, deps.Settings.DefaultCountryCode, denylist),
			Refresh:          command.NewRefresh(refreshTokens, users, issuer, denylist),
			Logout:           command.NewLogout(refreshTokens, deps.Auditor, denylist),
			UpdateProfile:    command.NewUpdateProfile(users, codes, pending, deps.Enqueuer, deps.Auditor, deps.Clock, deps.Settings.OTPLength, deps.Settings.OTPTTL, deps.Settings.DevMode, deps.Settings.DefaultCountryCode),
			ChangePassword:   command.NewChangePassword(users, deps.Hasher, refreshTokens, deps.Auditor, deps.Clock, denylist),
			SetupTOTP:        command.NewSetupTOTP(users, deps.Auditor, deps.Clock, deps.Settings.AppName),
			ConfirmTOTP:      command.NewConfirmTOTP(users, recovery, deps.Auditor, deps.Clock),
			DisableTOTP:      command.NewDisableTOTP(users, recovery, deps.Auditor, deps.Clock),
			VerifyMFA:        command.NewVerifyMFA(challenges, users, recovery, issuer, rateLimiter),
			SessionRevoke:    command.NewSessionRevoke(refreshTokens, deps.Auditor, denylist),
			Profile:          query.NewProfile(users, roles),
			Sessions:         query.NewSessions(refreshTokens, deps.Clock),
		},
		authn:         &authenticator{tokens: deps.Tokens, users: users, cache: deps.Cache},
		mailer:        deps.Mailer,
		sms:           deps.SMS,
		settings:      deps.Settings,
		clock:         deps.Clock,
		cache:         deps.Cache,
		processForgot: processForgot,
		processMagic:  processMagic,
	}
}

// Authenticator exposes the module's authentication implementation for the
// shared HTTP middleware.
func (m *Module) Authenticator() applicationauth.Authenticator { return m.authn }

// RegisterAPI mounts the module's JSON API routes on the shared router. The
// api.Deps bundle every use case, the authenticator and the public rate-limit
// settings.
func (m *Module) RegisterAPI(r chi.Router) {
	api.RegisterRoutes(r, api.Deps{
		Register:              m.API.Register,
		Login:                 m.API.Login,
		MagicLinkRequest:      m.API.MagicLinkRequest,
		MagicLinkVerify:       m.API.MagicLinkVerify,
		VerifyEmail:           m.API.VerifyEmail,
		VerifyPhone:           m.API.VerifyPhone,
		ForgotPassword:        m.API.ForgotPassword,
		ResetPassword:         m.API.ResetPassword,
		Refresh:               m.API.Refresh,
		Logout:                m.API.Logout,
		UpdateProfile:         m.API.UpdateProfile,
		ChangePassword:        m.API.ChangePassword,
		SetupTOTP:             m.API.SetupTOTP,
		ConfirmTOTP:           m.API.ConfirmTOTP,
		DisableTOTP:           m.API.DisableTOTP,
		VerifyMFA:             m.API.VerifyMFA,
		SessionRevoke:         m.API.SessionRevoke,
		Profile:               m.API.Profile,
		Sessions:              m.API.Sessions,
		Authenticator:         m.authn,
		Cache:                 m.cache,
		PublicRateLimitMax:    m.settings.PublicRateLimitMax,
		PublicRateLimitWindow: m.settings.PublicRateLimitWindow,
	})
}

// RegisterQueue registers the module's task handlers on a worker. Handlers
// render and send email/SMS using the module's branding and the worker-side
// processForgot/processMagic use cases.
func (m *Module) RegisterQueue(r appaqueue.Registrar) {
	queueadapter.Register(r, m.mailer, m.sms, queueadapterCommon(m.settings, m.clock), m.processForgot, m.processMagic)
}

// queueadapterCommon builds the branding injected into rendered messages.
func queueadapterCommon(s Settings, clk clock.Clock) queueadapter.Common {
	return queueadapter.Common{
		AppName:       s.AppName,
		BaseURL:       s.BaseURL,
		AssetsBaseURL: s.AssetsBaseURL,
		Year:          clk.Now().Year(),
	}
}

type authenticator struct {
	tokens token.Manager
	users  domain.UserRepository
	cache  cache.Cache
}

// Authenticate validates a raw access token and resolves it to an identity.
// It returns ErrUnauthenticated for malformed, revoked, or suspended-user
// tokens, and fails open on denylist cache errors (those are logged, not
// propagated, so an unavailable denylist never locks everyone out).
func (a *authenticator) Authenticate(ctx context.Context, raw string) (*applicationauth.Identity, error) {
	claims, err := a.tokens.ParseAccessToken(ctx, raw)
	if err != nil {
		return nil, applicationauth.ErrUnauthenticated
	}
	if a.cache != nil && claims.JTI != "" {
		if _, err := a.cache.Get(ctx, command.JTIDenyKey(claims.JTI)); err == nil {
			// The access token was explicitly revoked (logout, session revoke,
			// password change).
			return nil, applicationauth.ErrUnauthenticated
		} else if err != nil && !errors.Is(err, cache.ErrNotFound) {
			// Fail open on cache errors so an unavailable denylist does not
			// lock everyone out; refresh tokens are still revoked in the DB.
			slog.Warn("access token denylist check failed", "err", err)
		}
	}
	user, err := a.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, applicationauth.ErrUnauthenticated
	}
	if user == nil || user.IsSuspended() {
		return nil, applicationauth.ErrUnauthenticated
	}
	return &applicationauth.Identity{UserID: claims.UserID, Roles: claims.Roles}, nil
}

// rbacAdapter adapts the rbac module's Service to the narrow ports the auth
// module needs. It is nil-safe: without an RBAC service every method is a
// no-op, so the worker and tests can wire auth without RBAC.
type rbacAdapter struct {
	svc rbac.Service
}

// RolesAndPermissions implements port.Roles. When no RBAC service is wired it
// returns nil,nil,nil, so the caller treats the user as having no roles.
func (a rbacAdapter) RolesAndPermissions(ctx context.Context, userID string) ([]string, []string, error) {
	if a.svc == nil {
		return nil, nil, nil
	}
	roles, permissions, err := a.svc.RolesAndPermissions(ctx, userID)
	return roles, permissions, err
}

// AssignDefaultRole implements port.Roles. Without an RBAC service it is a
// no-op, so registration and seeding still succeed when RBAC is not wired.
func (a rbacAdapter) AssignDefaultRole(ctx context.Context, userID string) error {
	if a.svc == nil {
		return nil
	}
	return a.svc.AssignRole(ctx, userID, authorization.RoleUser)
}
