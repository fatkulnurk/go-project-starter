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
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/tasks"
)

// Common is the branding injected into every rendered message.
type Common = templates.Common

// Register wires every auth task handler onto the worker. common supplies the
// branding (app name, base URL, year) injected into every rendered message.
// processForgot and processMagic resolve delivery requests to accounts inside
// the worker, so unknown identifiers are skipped without sending.
func Register(r q.Registrar, m mailer.MailSender, s sms.Sender, common templates.Common, processForgot *commands.ProcessForgotPassword, processMagic *commands.ProcessMagicLink) {
	r.Register(tasks.SendVerificationEmail, sendVerificationEmail(m, common))
	r.Register(tasks.SendPhoneVerification, sendPhoneVerification(s, common))
	r.Register(tasks.SendForgotPasswordEmail, sendForgotPasswordEmail(m, common, processForgot))
	r.Register(tasks.SendForgotPasswordSMS, sendForgotPasswordSMS(s, common, processForgot))
	r.Register(tasks.SendMagicLinkEmail, sendMagicLinkEmail(m, common, processMagic))
}
