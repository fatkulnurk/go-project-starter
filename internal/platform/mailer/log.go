package mailer

import (
	"context"
	"log/slog"

	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
)

// Log prints messages to the logger instead of sending them. Useful in
// development so the starter runs without SMTP/SES credentials.
type Log struct{}

// NewLog builds a logging mail sender.
// It needs no configuration and is the default dev driver.
func NewLog() *Log { return &Log{} }

// Send implements mailer.MailSender.
// It logs the recipients, subject, plain-text body and attachment filenames
// and always returns nil — no message is actually delivered.
func (l *Log) Send(_ context.Context, msg mailer.Message) error {
	attachments := make([]string, 0, len(msg.Attachments))
	for _, a := range msg.Attachments {
		attachments = append(attachments, a.Filename)
	}
	slog.Info("mailer.send",
		"to", msg.To,
		"subject", msg.Subject,
		"text", msg.Text,
		"attachments", attachments,
	)
	return nil
}

// Close implements mailer.MailSender.
// There is nothing to release, so it always returns nil.
func (l *Log) Close() error { return nil }
