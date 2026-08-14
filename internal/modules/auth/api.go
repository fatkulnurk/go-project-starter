package auth

import (
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/queries"
)

// API is the public surface of the auth module. Other modules may depend only
// on this, never on the module's internals.
type API struct {
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
}
