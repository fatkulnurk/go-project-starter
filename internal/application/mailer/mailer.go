// Package mailer defines the cross-cutting mail contract. Business modules
// only know MailSender; the driver (log, smtp, ses) is chosen in the
// composition root.
package mailer

import "context"

// Attachment is a file attached to a Message.
type Attachment struct {
	Filename    string
	Content     []byte
	ContentType string // optional; inferred from the filename when empty
}

// Message is a single email to be delivered.
type Message struct {
	To          []string
	From        string
	FromName    string
	Subject     string
	Text        string
	HTML        string
	Attachments []Attachment
}

// MailSender delivers messages.
type MailSender interface {
	Send(ctx context.Context, msg Message) error
}
