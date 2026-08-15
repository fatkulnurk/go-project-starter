// Package queue contains the auth module's queue task handlers. It processes
// email/SMS tasks enqueued by the application layer. Handlers only call the
// mailer/sms application contracts; message copy is rendered from the
// embedded templates in the templates subpackage.
package queue

import (
	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
	q "github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/application/sms"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/command"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/task"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/template"
)

// Common is the branding injected into every rendered message.
type Common = template.Common

// Register wires every auth task handler onto the worker. common supplies the
// branding (app name, base URL, year) injected into every rendered message.
// processForgot and processMagic resolve delivery requests to accounts inside
// the worker, so unknown identifiers are skipped without sending.
func Register(r q.Registrar, m mailer.MailSender, s sms.Sender, common template.Common, processForgot *command.ProcessForgotPassword, processMagic *command.ProcessMagicLink) {
	r.Register(task.SendVerificationEmail, sendVerificationEmail(m, common))
	r.Register(task.SendPhoneVerification, sendPhoneVerification(s, common))
	r.Register(task.SendForgotPasswordEmail, sendForgotPasswordEmail(m, common, processForgot))
	r.Register(task.SendForgotPasswordSMS, sendForgotPasswordSMS(s, common, processForgot))
	r.Register(task.SendMagicLinkEmail, sendMagicLinkEmail(m, common, processMagic))
}
