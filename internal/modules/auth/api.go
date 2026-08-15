package auth

import (
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/command"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/query"
)

// API is the public surface of the auth module. Other modules may depend only
// on this, never on the module's internals.
type API struct {
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
}
