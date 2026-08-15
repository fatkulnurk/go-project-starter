// Package httpapi is the HTTP adapter of the auth module: it parses requests,
// calls one application use case, and renders responses. No business logic or
// SQL lives here.
package api

import (
	"context"
	"time"

	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/queries"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/go-chi/chi/v5"
)

// Deps bundles the application use cases and contracts the handlers need.
type Deps struct {
	Register         *commands.Register
	Login            *commands.Login
	MagicLinkRequest *commands.MagicLinkRequest
	MagicLinkVerify  *commands.MagicLinkVerify
	VerifyEmail      *commands.VerifyEmail
	VerifyPhone      *commands.VerifyPhone
	ForgotPassword   *commands.ForgotPassword
	ResetPassword    *commands.ResetPassword
	Refresh          *commands.Refresh
	Logout           *commands.Logout
	UpdateProfile    *commands.UpdateProfile
	Profile          *queries.Profile
	FindUserByEmail  *queries.FindUserByEmail
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
