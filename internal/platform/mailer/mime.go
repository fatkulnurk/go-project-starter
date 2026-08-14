package mailer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
)

// buildMIME renders a full RFC-822 message (headers + body) including any
// attachments as a multipart/mixed payload. Used by both the SMTP and SES
// drivers so attachment behavior is identical everywhere.
func buildMIME(from, fromName string, msg mailer.Message) ([]byte, error) {
	if len(msg.To) == 0 {
		return nil, fmt.Errorf("mail: no recipients")
	}
	fromAddr := from
	if fromName != "" {
		fromAddr = fmt.Sprintf("%s <%s>", encodeHeaderName(fromName), from)
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", fromAddr)
	fmt.Fprintf(&buf, "To: %s\r\n", encodeHeaderList(msg.To))
	fmt.Fprintf(&buf, "Subject: %s\r\n", encodeHeaderSubject(msg.Subject))
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")

	body := msg.Text
	contentType := "text/plain; charset=UTF-8"
	if msg.HTML != "" {
		body = msg.HTML
		contentType = "text/html; charset=UTF-8"
	}

	if len(msg.Attachments) == 0 {
		fmt.Fprintf(&buf, "Content-Type: %s\r\n\r\n", contentType)
		buf.WriteString(body)
		return buf.Bytes(), nil
	}

	mw := multipart.NewWriter(&buf)
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n\r\n", mw.Boundary())

	textPart, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type": {contentType},
	})
	if err != nil {
		return nil, err
	}
	if _, err := textPart.Write([]byte(body)); err != nil {
		return nil, err
	}

	for _, a := range msg.Attachments {
		if len(a.Content) == 0 {
			continue
		}
		ct := a.ContentType
		if ct == "" {
			ct = mime.TypeByExtension(filepath.Ext(a.Filename))
			if ct == "" {
				ct = "application/octet-stream"
			}
		}
		disp := mime.FormatMediaType("attachment", map[string]string{"filename": sanitizeHeaderValue(a.Filename)})
		part, err := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {ct},
			"Content-Transfer-Encoding": {"base64"},
			"Content-Disposition":       {disp},
		})
		if err != nil {
			return nil, err
		}
		enc := base64.NewEncoder(base64.StdEncoding, part)
		if _, err := enc.Write(a.Content); err != nil {
			enc.Close()
			return nil, err
		}
		if err := enc.Close(); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// sanitizeHeaderValue strips control characters that could terminate the
// header and inject further headers (header injection).
func sanitizeHeaderValue(v string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '\x00' {
			return -1
		}
		return r
	}, v)
}

// encodeHeaderName renders a display name, folding non-ASCII into RFC-2047
// encoded words.
func encodeHeaderName(name string) string {
	name = sanitizeHeaderValue(name)
	if isASCII(name) {
		return name
	}
	return mime.QEncoding.Encode("UTF-8", name)
}

// encodeHeaderSubject renders the subject with RFC-2047 encoding.
func encodeHeaderSubject(subject string) string {
	subject = sanitizeHeaderValue(subject)
	if isASCII(subject) {
		return subject
	}
	return mime.QEncoding.Encode("UTF-8", subject)
}

// encodeHeaderList renders a list of recipients, sanitizing each address.
func encodeHeaderList(addrs []string) string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, sanitizeHeaderValue(a))
	}
	return strings.Join(out, ", ")
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
