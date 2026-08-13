// Package queue contains the auth module's queue task handlers. It processes
// email/SMS tasks enqueued by the application layer. Handlers only call the
// mailer/sms application contracts.
package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
	q "github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/application/sms"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/tasks"
)

// Register wires every auth task handler onto the worker.
func Register(r q.Registrar, m mailer.MailSender, s sms.Sender) {
	r.Register(tasks.SendVerificationEmail, sendVerificationEmail(m))
	r.Register(tasks.SendPhoneVerification, sendPhoneVerification(s))
	r.Register(tasks.SendForgotPasswordEmail, sendForgotPasswordEmail(m))
	r.Register(tasks.SendForgotPasswordSMS, sendForgotPasswordSMS(s))
	r.Register(tasks.SendMagicLinkEmail, sendMagicLinkEmail(m))
}

func sendVerificationEmail(m mailer.MailSender) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p tasks.VerificationEmailPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode verification email payload: %v", q.ErrPermanent, err)
		}
		return m.Send(ctx, mailer.Message{
			To:      []string{p.To},
			Subject: "Verify your email",
			Text:    fmt.Sprintf("Hi %s, your email verification code is %s.", p.Name, p.Code),
			HTML:    fmt.Sprintf("<p>Hi %s,</p><p>Your email verification code is <b>%s</b>.</p>", p.Name, p.Code),
		})
	}
}

func sendPhoneVerification(s sms.Sender) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p tasks.PhoneVerificationPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode phone verification payload: %v", q.ErrPermanent, err)
		}
		return s.Send(ctx, sms.Message{
			To:   p.To,
			Body: fmt.Sprintf("Your phone verification code is %s.", p.Code),
		})
	}
}

func sendForgotPasswordEmail(m mailer.MailSender) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p tasks.ForgotPasswordEmailPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode forgot-password email payload: %v", q.ErrPermanent, err)
		}
		return m.Send(ctx, mailer.Message{
			To:      []string{p.To},
			Subject: "Reset your password",
			Text:    fmt.Sprintf("Hi %s, your password reset code is %s.", p.Name, p.Code),
			HTML:    fmt.Sprintf("<p>Hi %s,</p><p>Your password reset code is <b>%s</b>.</p>", p.Name, p.Code),
		})
	}
}

func sendForgotPasswordSMS(s sms.Sender) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p tasks.ForgotPasswordSMSPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode forgot-password sms payload: %v", q.ErrPermanent, err)
		}
		return s.Send(ctx, sms.Message{
			To:   p.To,
			Body: fmt.Sprintf("Your password reset code is %s.", p.Code),
		})
	}
}

func sendMagicLinkEmail(m mailer.MailSender) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p tasks.MagicLinkEmailPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode magic link payload: %v", q.ErrPermanent, err)
		}
		return m.Send(ctx, mailer.Message{
			To:      []string{p.To},
			Subject: "Your magic sign-in link",
			Text:    fmt.Sprintf("Hi %s, sign in with this link: %s", p.Name, p.Link),
			HTML:    fmt.Sprintf("<p>Hi %s,</p><p><a href=\"%s\">Sign in with this link</a>.</p>", p.Name, p.Link),
		})
	}
}
