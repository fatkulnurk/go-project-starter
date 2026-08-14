// Package auth is the authentication module: registration, login (password
// and magic link), email/phone verification, forgot/reset password and token
// management. It depends only on cross-cutting application contracts and the
// repositories wired through Dependencies.
package auth

import (
	"context"
	"database/sql"
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
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/adapters/httpapi"
	queueadapter "github.com/fatkulnurk/go-project-starter/internal/modules/auth/adapters/queue"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/queries"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/infrastructure"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
	"github.com/go-chi/chi/v5"
)

// Settings carries auth behavior configured in the composition root.
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
	DevMode       bool
}

// Dependencies are the ports the module needs; all wired by the composition
// root.
type Dependencies struct {
	DB       *sql.DB
	DBDriver string
	Cache    cache.Cache
	Enqueuer appaqueue.Enqueuer
	Mailer   mailer.MailSender
	SMS      sms.Sender
	Tokens   token.Manager
	Hasher   hash.PasswordHasher
	Clock    clock.Clock
	RBAC     rbac.Service
	Auditor  audit.Auditor
	Settings Settings
	// Location is the app timezone for SQL-written timestamps.
	Location *time.Location
}

// Module wires the auth use cases and their adapters.
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
	processForgot *commands.ProcessForgotPassword
	processMagic  *commands.ProcessMagicLink
}

// New constructs the auth module.
func New(deps Dependencies) *Module {
	users := infrastructure.NewUserRepository(deps.DB, deps.DBDriver, deps.Location)
	refreshTokens := infrastructure.NewRefreshTokenRepository(deps.DB, deps.DBDriver, deps.Location)
	codes := infrastructure.NewVerificationCodeRepository(deps.DB, deps.DBDriver, deps.Location)
	pending := infrastructure.NewPendingContactChangeRepository(deps.DB, deps.DBDriver, deps.Location)

	roles := rbacAdapter{svc: deps.RBAC}

	issuer := commands.NewTokenIssuer(deps.Tokens, refreshTokens, roles, deps.Auditor, deps.Settings.AccessTokenTTL, deps.Settings.RefreshTokenTTL, deps.Clock)
	rateLimiter := commands.NewLoginRateLimiter(deps.Cache, deps.Settings.RateLimitMax, deps.Settings.RateLimitWindow)
	forgotLimiter := commands.NewRateLimiter(deps.Cache, deps.Settings.RateLimitMax, deps.Settings.RateLimitWindow, "rl:forgot")
	magicLimiter := commands.NewRateLimiter(deps.Cache, deps.Settings.RateLimitMax, deps.Settings.RateLimitWindow, "rl:magic")

	processForgot := commands.NewProcessForgotPassword(users, codes, deps.Settings.OTPLength, deps.Settings.OTPTTL, deps.Clock)
	processMagic := commands.NewProcessMagicLink(users, codes, deps.Settings.BaseURL, deps.Settings.MagicLinkTTL, deps.Clock)

	return &Module{
		API: API{
			Register:         commands.NewRegister(users, codes, deps.Hasher, deps.Enqueuer, roles, deps.Auditor, deps.Clock, deps.Settings.OTPLength, deps.Settings.OTPTTL, deps.Settings.OTPMaxAttempts, deps.Settings.DevMode),
			Login:            commands.NewLogin(users, deps.Hasher, issuer, deps.Settings.RequireEmailVerified, rateLimiter),
			MagicLinkRequest: commands.NewMagicLinkRequest(deps.Enqueuer, deps.Settings.MagicLinkTTL, magicLimiter),
			MagicLinkVerify:  commands.NewMagicLinkVerify(codes, users, issuer),
			VerifyEmail:      commands.NewVerifyEmail(users, codes, pending, deps.Auditor, deps.Clock, deps.Settings.OTPMaxAttempts),
			VerifyPhone:      commands.NewVerifyPhone(users, codes, pending, deps.Auditor, deps.Clock, deps.Settings.OTPMaxAttempts),
			ForgotPassword:   commands.NewForgotPassword(deps.Enqueuer, deps.Settings.OTPTTL, forgotLimiter),
			ResetPassword:    commands.NewResetPassword(users, codes, refreshTokens, deps.Hasher, deps.Auditor, deps.Clock, deps.Settings.OTPMaxAttempts),
			Refresh:          commands.NewRefresh(refreshTokens, users, issuer),
			Logout:           commands.NewLogout(refreshTokens, deps.Auditor),
			UpdateProfile:    commands.NewUpdateProfile(users, codes, pending, deps.Enqueuer, deps.Auditor, deps.Clock, deps.Settings.OTPLength, deps.Settings.OTPTTL, deps.Settings.DevMode),
			Profile:          queries.NewProfile(users, roles),
			FindUserByEmail:  queries.NewFindUserByEmail(users),
		},
		authn:         &authenticator{tokens: deps.Tokens, users: users},
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

// RegisterHTTP mounts the module's routes on the shared router.
func (m *Module) RegisterHTTP(r chi.Router) {
	httpapi.RegisterRoutes(r, httpapi.Deps{
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
		Profile:               m.API.Profile,
		FindUserByEmail:       m.API.FindUserByEmail,
		Authenticator:         m.authn,
		Authorizer:            authorization.AllowAll{},
		Cache:                 m.cache,
		PublicRateLimitMax:    m.settings.PublicRateLimitMax,
		PublicRateLimitWindow: m.settings.PublicRateLimitWindow,
	})
}

// RegisterQueue registers the module's task handlers on a worker.
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
}

func (a *authenticator) Authenticate(ctx context.Context, raw string) (*applicationauth.Identity, error) {
	claims, err := a.tokens.ParseAccessToken(ctx, raw)
	if err != nil {
		return nil, applicationauth.ErrUnauthenticated
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

// RolesAndPermissions implements ports.Roles.
func (a rbacAdapter) RolesAndPermissions(ctx context.Context, userID string) ([]string, []string, error) {
	if a.svc == nil {
		return nil, nil, nil
	}
	roles, permissions, err := a.svc.RolesAndPermissions(ctx, userID)
	return roles, permissions, err
}

// AssignDefaultRole implements ports.Roles.
func (a rbacAdapter) AssignDefaultRole(ctx context.Context, userID string) error {
	if a.svc == nil {
		return nil
	}
	return a.svc.AssignRole(ctx, userID, authorization.RoleUser)
}
