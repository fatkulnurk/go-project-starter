package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
	q "github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/application/sms"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/command"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/task"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/template"
)

func sendVerificationEmail(m mailer.MailSender, common template.Common) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p task.VerificationEmailPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode verification email payload: %v", q.ErrPermanent, err)
		}
		subject, text, html, err := template.Email("email_verification", template.EmailVerificationData{
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

func sendPhoneVerification(s sms.Sender, common template.Common) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p task.PhoneVerificationPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode phone verification payload: %v", q.ErrPermanent, err)
		}
		body, err := template.SMS("sms_verification", template.SMSVerificationData{
			Common: common,
			Code:   p.Code,
		})
		if err != nil {
			return fmt.Errorf("%w: render phone verification sms: %v", q.ErrPermanent, err)
		}
		return s.Send(ctx, sms.Message{To: p.To, Body: body})
	}
}

func sendForgotPasswordEmail(m mailer.MailSender, common template.Common, process *command.ProcessForgotPassword) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p task.ForgotPasswordEmailPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode forgot-password email payload: %v", q.ErrPermanent, err)
		}
		res, err := process.Execute(ctx, p.Identifier)
		if err != nil {
			return err
		}
		if res.User == nil {
			// Unknown identifier: deliver nothing, silently.
			return nil
		}
		subject, text, html, err := template.Email("email_forgot_password", template.EmailForgotPasswordData{
			Common: common,
			Name:   res.User.Name,
			Code:   res.Code,
		})
		if err != nil {
			return fmt.Errorf("%w: render forgot-password email: %v", q.ErrPermanent, err)
		}
		return m.Send(ctx, mailer.Message{To: []string{*res.User.Email}, Subject: subject, Text: text, HTML: html})
	}
}

func sendForgotPasswordSMS(s sms.Sender, common template.Common, process *command.ProcessForgotPassword) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p task.ForgotPasswordSMSPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode forgot-password sms payload: %v", q.ErrPermanent, err)
		}
		res, err := process.Execute(ctx, p.Identifier)
		if err != nil {
			return err
		}
		if res.User == nil {
			return nil
		}
		body, err := template.SMS("sms_forgot_password", template.SMSForgotPasswordData{
			Common: common,
			Code:   res.Code,
		})
		if err != nil {
			return fmt.Errorf("%w: render forgot-password sms: %v", q.ErrPermanent, err)
		}
		return s.Send(ctx, sms.Message{To: *res.User.Phone, Body: body})
	}
}

func sendMagicLinkEmail(m mailer.MailSender, common template.Common, process *command.ProcessMagicLink) q.TaskHandler {
	return func(ctx context.Context, payload []byte) error {
		var p task.MagicLinkEmailPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: decode magic link payload: %v", q.ErrPermanent, err)
		}
		res, err := process.Execute(ctx, p.Email)
		if err != nil {
			return err
		}
		if res.User == nil {
			return nil
		}
		subject, text, html, err := template.Email("email_magic_link", template.EmailMagicLinkData{
			Common: common,
			Name:   res.User.Name,
			Link:   res.Link,
		})
		if err != nil {
			return fmt.Errorf("%w: render magic link email: %v", q.ErrPermanent, err)
		}
		return m.Send(ctx, mailer.Message{To: []string{*res.User.Email}, Subject: subject, Text: text, HTML: html})
	}
}
