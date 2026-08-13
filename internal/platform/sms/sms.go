// Package sms provides SMS driver implementations and a factory.
package sms

import (
	"fmt"

	"github.com/fatkulnurk/go-project-starter/internal/application/sms"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// New returns an sms.Sender for the configured driver.
func New(cfg config.SMSConfig) (sms.Sender, error) {
	switch cfg.Driver {
	case "log":
		return NewLog(), nil
	case "twilio":
		return NewTwilio(cfg.From, cfg.Twilio)
	default:
		return nil, fmt.Errorf("unknown sms driver %q", cfg.Driver)
	}
}
