package mailer

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// SMTP sends email over a plain SMTP server (net/smtp).
type SMTP struct {
	from     string
	fromName string
	host     string
	port     int
	user     string
	password string
}

// NewSMTP builds an SMTP mail sender.
func NewSMTP(from, fromName string, cfg config.SMTPConfig) (*SMTP, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("SMTP_HOST is required")
	}
	return &SMTP{from: from, fromName: fromName, host: cfg.Host, port: cfg.Port, user: cfg.User, password: cfg.Password}, nil
}

// Send implements mailer.MailSender. Supports text, HTML and attachments.
func (s *SMTP) Send(_ context.Context, msg mailer.Message) error {
	data, err := buildMIME(s.from, s.fromName, msg)
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	return smtp.SendMail(addr, auth, s.from, msg.To, data)
}
