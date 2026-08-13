package sms

import (
	"context"
	"log/slog"

	"github.com/fatkulnurk/go-project-starter/internal/application/sms"
)

// Log prints messages to the logger instead of sending them. Default driver
// so the starter runs without a Twilio account.
type Log struct{}

// NewLog builds a logging SMS sender.
func NewLog() *Log { return &Log{} }

// Send implements sms.Sender.
func (l *Log) Send(_ context.Context, msg sms.Message) error {
	slog.Info("sms.send",
		"to", msg.To,
		"body", msg.Body,
	)
	return nil
}
