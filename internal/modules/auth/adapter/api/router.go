// Package httpapi is the HTTP adapter of the auth module: it parses requests,
// calls one application use case, and renders responses. No business logic or
// SQL lives here.
package api

import (
	"context"
	"time"

	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/command"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/query"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/go-chi/chi/v5"
)

// Deps bundles the application use cases and contracts the handlers need.
// Cache and the public rate-limit values protect the unauthenticated endpoints.
type Deps struct {
	Register         *command.Register
	Login            *command.Login
	MagicLinkRequest *command.MagicLinkRequest
	MagicLinkVerify  *command.MagicLinkVerify
	VerifyEmail      *command.VerifyEmail
	VerifyPhone      *command.VerifyPhone
	ForgotPassword   *command.ForgotPassword
	ResetPassword    *command.ResetPassword
	Refresh          *command.Refresh
	Logout           *command.Logout
	UpdateProfile    *command.UpdateProfile
	ChangePassword   *command.ChangePassword
	SetupTOTP        *command.SetupTOTP
	ConfirmTOTP      *command.ConfirmTOTP
	DisableTOTP      *command.DisableTOTP
	VerifyMFA        *command.VerifyMFA
	SessionRevoke    *command.SessionRevoke
	Profile          *query.Profile
	Sessions         *query.Sessions
	Authenticator    appauth.Authenticator
	// Cache + rate limits protect the unauthenticated endpoints against
	// brute force and message-bombing abuse.
	Cache                 cache.Cache
	PublicRateLimitMax    int64
	PublicRateLimitWindow time.Duration
}

// RegisterRoutes mounts the auth API under /api/v1/auth, split into a public,
// rate-limited group and an authenticated group. deps supplies every use case
// plus the cache backing the public rate limits.
func RegisterRoutes(r chi.Router, deps Deps) {
	h := &handler{deps: deps}

	r.Route("/api/v1/auth", func(r chi.Router) {
		public := platformhttp.RateLimitByIP(deps.Cache, deps.PublicRateLimitMax, deps.PublicRateLimitWindow)
		r.Group(func(r chi.Router) {
			r.Use(public)
			r.Post("/register", h.register)
			r.Post("/login", h.login)
			r.Post("/mfa/verify", h.verifyMFA)
			r.Post("/magic-link", h.magicLinkRequest)
			r.Post("/magic-link/verify", h.magicLinkVerify)
			r.Get("/magic-link/verify", h.magicLinkVerifyGet)
			r.Post("/verify-email", h.verifyEmail)
			r.Post("/verify-phone", h.verifyPhone)
			r.Post("/forgot-password", h.forgotPassword)
			r.Post("/reset-password", h.resetPassword)
			r.Post("/refresh", h.refresh)
		})

		r.Group(func(r chi.Router) {
			r.Use(platformhttp.AuditActor)
			r.Use(platformhttp.Authenticate(deps.Authenticator))
			r.Post("/logout", h.logout)
			r.Get("/me", h.me)
			r.Patch("/me", h.updateProfile)
			r.Post("/me/password", h.changePassword)
			r.Get("/sessions", h.sessions)
			r.Delete("/sessions/{familyID}", h.revokeSession)
			r.Post("/mfa/setup", h.setupTOTP)
			r.Post("/mfa/confirm", h.confirmTOTP)
			r.Post("/mfa/disable", h.disableTOTP)
		})
	})
}

// identity returns the authenticated identity from the request context. It
// returns ErrUnauthenticated when no identity is present.
func identity(ctx context.Context) (*appauth.Identity, error) {
	id := appauth.IdentityFrom(ctx)
	if id == nil {
		return nil, appauth.ErrUnauthenticated
	}
	return id, nil
}
