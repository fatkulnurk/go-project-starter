// Package templates renders the auth module's outbound messages (email HTML +
// plain text, SMS) from embedded Go template. Email HTML is composed by
// attaching this module's content templates to the shared layout owned by
// internal/platform/mailer; text and SMS are rendered by dedicated text
// template. Channel templates live under email/ and sms/ subfolders.
//
// Templates are parsed once at startup via template.Must, so a syntax error
// fails the process immediately instead of surfacing on first send.
package template

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	htmltemplate "html/template"

	"github.com/fatkulnurk/go-project-starter/internal/application/branding"
	"github.com/fatkulnurk/go-project-starter/internal/platform/mailer"
)

//go:embed email/*.html email/*.txt sms/*.txt
var files embed.FS

// Common is the branding embedded by every view model. It aliases the mailer
// layout data so templates read {{.Common.AppName}} unchanged.
type Common = mailer.Common

// EmailVerificationData is the view model for email_verification: the brand
// common, the recipient's name and the numeric code.
type EmailVerificationData struct {
	Common
	Name string
	Code string
}

// EmailForgotPasswordData is the view model for email_forgot_password: the
// brand common, the recipient's name and the reset code.
type EmailForgotPasswordData struct {
	Common
	Name string
	Code string
}

// EmailMagicLinkData is the view model for email_magic_link: the brand common,
// the recipient's name and the one-time login link.
type EmailMagicLinkData struct {
	Common
	Name string
	Link string
}

// SMSVerificationData is the view model for sms_verification: the brand common
// and the numeric code.
type SMSVerificationData struct {
	Common
	Code string
}

// SMSForgotPasswordData is the view model for sms_forgot_password: the brand
// common and the reset code.
type SMSForgotPasswordData struct {
	Common
	Code string
}

// emailTmpls holds one parsed template tree per email: the shared platform
// layout plus this module's content file. A content block named "content"
// stays unique within its tree, so emails never overwrite each other.
var emailTmpls = map[string]*htmltemplate.Template{
	"email_verification":    mustEmail("email_verification"),
	"email_forgot_password": mustEmail("email_forgot_password"),
	"email_magic_link":      mustEmail("email_magic_link"),
}

// textTmpl carries every plain-text message and subject in one set (unique
// definition names, no shared blocks).
var textTmpl = template.Must(template.ParseFS(files, "email/*.txt", "sms/*.txt"))

// mustEmail parses the shared platform email layout, then attaches the given
// content template from this module.
func mustEmail(name string) *htmltemplate.Template {
	layout, err := mailer.NewEmailLayout()
	if err != nil {
		panic(err)
	}
	return htmltemplate.Must(layout.ParseFS(files, "email/"+name+".html"))
}

// Email renders subject, plain text and HTML for the named email template
// (e.g. "email_verification"). data is the view model; it embeds Common.
func Email(name string, data any) (subject, text, html string, err error) {
	if subject, err = branding.Render(textTmpl, name+"_subject", data); err != nil {
		return "", "", "", err
	}
	if text, err = branding.Render(textTmpl, name+"_text", data); err != nil {
		return "", "", "", err
	}
	tmpl, ok := emailTmpls[name]
	if !ok {
		return "", "", "", fmt.Errorf("unknown email template %q", name)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", "", "", err
	}
	return subject, text, buf.String(), nil
}

// SMS renders the body of the named SMS template (e.g. "sms_verification")
// from the shared text-template set. data is the matching view model.
func SMS(name string, data any) (string, error) {
	return branding.Render(textTmpl, name, data)
}
