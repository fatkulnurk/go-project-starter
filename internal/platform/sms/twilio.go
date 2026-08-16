package sms

import (
	"context"
	"fmt"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/sms"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// Twilio sends SMS through the Twilio Messages API.
// It uses either a messaging service or a From number depending on the config
// and per-message overrides.
type Twilio struct {
	from         string
	accountSID   string
	authToken    string
	messagingSID string
	client       *twilio.RestClient
}

// NewTwilio builds a Twilio SMS sender.
// It returns an error when the account SID or auth token is missing; the
// client enforces a 30s timeout on API calls.
func NewTwilio(from string, cfg config.TwilioConfig) (*Twilio, error) {
	if cfg.AccountSID == "" || cfg.AuthToken == "" {
		return nil, fmt.Errorf("TWILIO_ACCOUNT_SID and TWILIO_AUTH_TOKEN are required")
	}
	client := twilio.NewRestClientWithParams(twilio.ClientParams{Username: cfg.AccountSID, Password: cfg.AuthToken})
	client.SetTimeout(30 * time.Second)
	return &Twilio{
		from:         from,
		accountSID:   cfg.AccountSID,
		authToken:    cfg.AuthToken,
		messagingSID: cfg.MessagingSID,
		client:       client,
	}, nil
}

// Send implements sms.Sender.
// The context is ignored. The sender is the messaging service when configured,
// else msg.From, else the default from; errors are wrapped with the recipient.
func (t *Twilio) Send(ctx context.Context, msg sms.Message) error {
	params := &openapi.CreateMessageParams{}
	params.SetTo(msg.To)
	params.SetBody(msg.Body)
	switch {
	case t.messagingSID != "":
		params.SetMessagingServiceSid(t.messagingSID)
	case msg.From != "":
		params.SetFrom(msg.From)
	default:
		params.SetFrom(t.from)
	}
	if _, err := t.client.Api.CreateMessage(params); err != nil {
		return fmt.Errorf("twilio send to %s: %w", msg.To, err)
	}
	return nil
}
