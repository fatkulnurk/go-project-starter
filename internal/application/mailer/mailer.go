// Package mailer defines the cross-cutting mail contract. Business modules
// only know MailSender; the driver (log, smtp, ses) is chosen in the
// composition root.
package mailer

import "context"

// Attachment is a file attached to a Message: its user-facing filename, the
// raw content, and an optional Content-Type inferred from the filename when
// empty.
type Attachment struct {
	Filename    string
	Content     []byte
	ContentType string // optional; inferred from the filename when empty
}

// Message is a single email to be delivered. To lists the recipients; From
// and FromName override the driver defaults when set; either Text or HTML is
// the body (HTML wins when both are present).
type Message struct {
	To          []string
	From        string
	FromName    string
	Subject     string
	Text        string
	HTML        string
	Attachments []Attachment
}

// MailSender delivers messages. Close releases any pooled resources (SMTP
// connections); drivers without resources return nil.
type MailSender interface {
	// Send delivers one message to all recipients. It returns an error when
	// the driver rejects the message (bad address, network failure, ...).
	Send(ctx context.Context, msg Message) error

	// Close releases pooled resources held by the driver. It is safe to call
	// once at shutdown and is a no-op for stateless drivers.
	Close() error
}
