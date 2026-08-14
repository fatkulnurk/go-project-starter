package mailer

import (
	"strings"
	"testing"

	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
)

// headerLines returns the header block of the message.
func headerBlock(data []byte) string {
	s := string(data)
	if i := strings.Index(s, "\r\n\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}

func TestBuildMIMESanitizesHeaders(t *testing.T) {
	msg := mailer.Message{
		To:      []string{"victim@example.com\r\nBcc: attacker@example.com"},
		Subject: "Hello\r\nBcc: attacker@example.com",
		Text:    "body",
		Attachments: []mailer.Attachment{
			{Filename: "evil\r\nX-Injected: 1.txt", Content: []byte("x")},
		},
	}
	data, err := buildMIME("from@example.com", "Injected\r\nX-Evil: 1", msg)
	if err != nil {
		t.Fatalf("buildMIME error: %v", err)
	}
	for _, line := range strings.Split(headerBlock(data), "\r\n") {
		for _, injected := range []string{"Bcc:", "X-Injected:", "X-Evil:"} {
			if strings.HasPrefix(line, injected) {
				t.Fatalf("header injection produced standalone header line %q:\n%s", line, data)
			}
		}
	}
}

func TestBuildMIMEEscapesUTF8Subject(t *testing.T) {
	data, err := buildMIME("from@example.com", "", mailer.Message{
		To:      []string{"a@example.com"},
		Subject: "Příliš žluťoučký kůň",
		Text:    "body",
	})
	if err != nil {
		t.Fatalf("buildMIME error: %v", err)
	}
	if !strings.Contains(strings.ToLower(string(data)), "=?utf-8?q?") {
		t.Fatalf("UTF-8 subject not encoded: %s", data)
	}
}
