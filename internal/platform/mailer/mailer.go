// Package mailer provides mail driver implementations and a factory.
package mailer

import (
	"fmt"

	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// New returns a mailer.MailSender for the configured driver.
// Unsupported drivers return an error; callers should Close the sender once
// the application shuts down.
func New(cfg config.MailConfig) (mailer.MailSender, error) {
	switch cfg.Driver {
	case config.DriverLog:
		return NewLog(), nil
	case config.DriverSMTP:
		return NewSMTP(cfg.From, cfg.FromName, cfg.SMTP)
	case config.DriverSES:
		return NewSES(cfg.From, cfg.FromName, cfg.SES)
	default:
		return nil, fmt.Errorf("unknown mail driver %q", cfg.Driver)
	}
}
