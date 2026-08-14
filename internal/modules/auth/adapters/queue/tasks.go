// Package queue contains the auth module's queue task handlers. It processes
// email/SMS tasks enqueued by the application layer. Handlers only call the
// mailer/sms application contracts; message copy is rendered from the
// embedded templates in the templates subpackage.
package queue

import (
	"context"
	"encoding/json"
	"fmt"

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

func sendVerificationEmail(m mailer.MailSender, common templates.Common) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p tasks.VerificationEmailPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode verification email payload: %v", q.ErrPermanent, err)
		}
		subject, text, html, err := templates.Email("email_verification", templates.EmailVerificationData{
			Common: common,
			Name:   p.Name,
			Code:   p.Code,
		})
		if err != nil {
			return fmt.Errorf("%w: render verification email: %v", q.ErrPermanent, err)
		}
		return m.Send(ctx, mailer.Message{To: []string{p.To}, Subject: subject, Text: text, HTML: html})
	}
}

func sendPhoneVerification(s sms.Sender, common templates.Common) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p tasks.PhoneVerificationPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode phone verification payload: %v", q.ErrPermanent, err)
		}
		body, err := templates.SMS("sms_verification", templates.SMSVerificationData{
			Common: common,
			Code:   p.Code,
		})
		if err != nil {
			return fmt.Errorf("%w: render phone verification sms: %v", q.ErrPermanent, err)
		}
		return s.Send(ctx, sms.Message{To: p.To, Body: body})
	}
}

func sendForgotPasswordEmail(m mailer.MailSender, common templates.Common) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p tasks.ForgotPasswordEmailPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode forgot-password email payload: %v", q.ErrPermanent, err)
		}
		subject, text, html, err := templates.Email("email_forgot_password", templates.EmailForgotPasswordData{
			Common: common,
			Name:   p.Name,
			Code:   p.Code,
		})
		if err != nil {
			return fmt.Errorf("%w: render forgot-password email: %v", q.ErrPermanent, err)
		}
		return m.Send(ctx, mailer.Message{To: []string{p.To}, Subject: subject, Text: text, HTML: html})
	}
}

func sendForgotPasswordSMS(s sms.Sender, common templates.Common) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p tasks.ForgotPasswordSMSPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode forgot-password sms payload: %v", q.ErrPermanent, err)
		}
		body, err := templates.SMS("sms_forgot_password", templates.SMSForgotPasswordData{
			Common: common,
			Code:   p.Code,
		})
		if err != nil {
			return fmt.Errorf("%w: render forgot-password sms: %v", q.ErrPermanent, err)
		}
		return s.Send(ctx, sms.Message{To: p.To, Body: body})
	}
}

func sendMagicLinkEmail(m mailer.MailSender, common templates.Common) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p tasks.MagicLinkEmailPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode magic link payload: %v", q.ErrPermanent, err)
		}
		subject, text, html, err := templates.Email("email_magic_link", templates.EmailMagicLinkData{
			Common: common,
			Name:   p.Name,
			Link:   p.Link,
		})
		if err != nil {
			return fmt.Errorf("%w: render magic link email: %v", q.ErrPermanent, err)
		}
		return m.Send(ctx, mailer.Message{To: []string{p.To}, Subject: subject, Text: text, HTML: html})
	}
}
