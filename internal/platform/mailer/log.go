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
func NewLog() *Log { return &Log{} }

// Send implements mailer.MailSender.
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
