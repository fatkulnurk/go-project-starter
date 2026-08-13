// Package sms defines the cross-cutting SMS contract. Business modules only
// know Sender; the driver (log, twilio) is chosen in the composition root.
package sms

import "context"

// Message is a single SMS to be delivered.
type Message struct {
	To   string
	From string
	Body string
}

// Sender delivers SMS messages.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
