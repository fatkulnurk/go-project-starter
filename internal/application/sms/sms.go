// Package sms defines the cross-cutting SMS contract. Business modules only
// know Sender; the driver (log, twilio) is chosen in the composition root.
package sms

import "context"

// Message is a single SMS to be delivered: the recipient, an optional sender
// id, and the message body.
type Message struct {
	To   string
	From string
	Body string
}

// Sender delivers SMS messages.
// The concrete driver (log, twilio) is chosen in the composition root, so
// business modules only depend on this contract.
type Sender interface {
	// Send delivers one message. It returns an error when the backend rejects
	// the send (e.g. invalid credentials, network failure).
	Send(ctx context.Context, msg Message) error
}
