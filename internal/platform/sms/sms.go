// Package sms provides SMS driver implementations and a factory.
package sms

import (
	"fmt"

	"github.com/fatkulnurk/go-project-starter/internal/application/sms"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// New returns an sms.Sender for the configured driver.
// Unsupported drivers return an error; the log driver needs no credentials.
func New(cfg config.SMSConfig) (sms.Sender, error) {
	switch cfg.Driver {
	case config.DriverLog:
		return NewLog(), nil
	case config.DriverTwilio:
		return NewTwilio(cfg.From, cfg.Twilio)
	default:
		return nil, fmt.Errorf("unknown sms driver %q", cfg.Driver)
	}
}
