// Package branding carries the shared branding and template-render helper used
// by every module that renders user-facing copy (web pages, email, SMS).
// Keeping it here — in the application layer — gives web and email layouts one
// source of truth for the brand data instead of duplicating a Common struct
// in each platform package.
package branding

import (
	"bytes"
	"io"
)

// Common is the branding embedded by every view model. Templates read it as
// {{.Common.AppName}}, {{.Common.BaseURL}}, {{.Common.AssetsBaseURL}} and
// {{.Common.Year}}.
type Common struct {
	AppName string
	BaseURL string
	// AssetsBaseURL is the absolute base URL of static assets (e.g. a CDN or
	// the app's /assets mount).
	AssetsBaseURL string
	Year          int
}

// templateExecutor is satisfied by both html/template.Template and
// text/template.Template, so Render works with either template set.
type templateExecutor interface {
	// ExecuteTemplate renders the named template into w with the given data.
	ExecuteTemplate(w io.Writer, name string, data any) error
}

// Render executes the template named by tn with data and returns the output.
// It works for both HTML and text template sets.
func Render(exec templateExecutor, tn string, data any) (string, error) {
	var buf bytes.Buffer
	if err := exec.ExecuteTemplate(&buf, tn, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
