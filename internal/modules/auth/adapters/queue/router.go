// Package queue contains the auth module's queue task handlers. It processes
// email/SMS tasks enqueued by the application layer. Handlers only call the
// mailer/sms application contracts; message copy is rendered from the
// embedded templates in the templates subpackage.
package queue

import (
	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
	q "github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/application/sms"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/adapters/queue/templates"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/tasks"
)

// Common is the branding injected into every rendered message.
type Common = templates.Common

// Register wires every auth task handler onto the worker. common supplies the
// branding (app name, base URL, year) injected into every rendered message.
func Register(r q.Registrar, m mailer.MailSender, s sms.Sender, common templates.Common) {
	r.Register(tasks.SendVerificationEmail, sendVerificationEmail(m, common))
	r.Register(tasks.SendPhoneVerification, sendPhoneVerification(s, common))
	r.Register(tasks.SendForgotPasswordEmail, sendForgotPasswordEmail(m, common))
	r.Register(tasks.SendForgotPasswordSMS, sendForgotPasswordSMS(s, common))
	r.Register(tasks.SendMagicLinkEmail, sendMagicLinkEmail(m, common))
}
