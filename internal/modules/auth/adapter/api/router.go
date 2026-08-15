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
	Profile          *query.Profile
	FindUserByEmail  *query.FindUserByEmail
	Authenticator    appauth.Authenticator
	// Cache + rate limits protect the unauthenticated endpoints against
	// brute force and message-bombing abuse.
	Cache                 cache.Cache
	PublicRateLimitMax    int64
	PublicRateLimitWindow time.Duration
}

// RegisterRoutes mounts the auth API under /api/v1/auth.
func RegisterRoutes(r chi.Router, deps Deps) {
	h := &handler{deps: deps}

	r.Route("/api/v1/auth", func(r chi.Router) {
		public := platformhttp.RateLimitByIP(deps.Cache, deps.PublicRateLimitMax, deps.PublicRateLimitWindow)
		r.Group(func(r chi.Router) {
			r.Use(public)
			r.Post("/register", h.register)
			r.Post("/login", h.login)
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
			r.Use(platformhttp.Authenticate(deps.Authenticator))
			r.Post("/logout", h.logout)
			r.Get("/me", h.me)
			r.Patch("/me", h.updateProfile)
		})
	})
}

// identity extracts the authenticated identity or returns an error.
func identity(ctx context.Context) (*appauth.Identity, error) {
	id := appauth.IdentityFrom(ctx)
	if id == nil {
		return nil, appauth.ErrUnauthenticated
	}
	return id, nil
}
