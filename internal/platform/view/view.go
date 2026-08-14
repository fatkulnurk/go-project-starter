// Package view renders HTML pages (browser views) from embedded Go templates.
// It owns the shared base layout — the "how" of every page — while business
// modules attach their own content templates via ParseFS, mirroring the email
// layout in internal/platform/mailer.
package view

import (
	"bytes"
	"embed"

	htmltemplate "html/template"
)

//go:embed base.html
var baseFS embed.FS

// Common carries branding shared by the base layout. Module page templates
// embed it so the layout can render the header and footer.
type Common struct {
	AppName string
	BaseURL string
	// AssetsBaseURL is the absolute base URL of static assets (e.g. a CDN or
	// the app's /assets mount).
	AssetsBaseURL string
	Year          int
}

// NewLayout parses the shared base layout into a template set that defines
// the "layout" shell and a "content" block. Modules attach their own content
// templates to it with (*html/template.Template).ParseFS.
func NewLayout() (*htmltemplate.Template, error) {
	return htmltemplate.ParseFS(baseFS, "base.html")
}

// Render executes the template named by tn (a module template attached to the
// base layout) and returns the resulting HTML.
func Render(t *htmltemplate.Template, tn string, data any) (string, error) {
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, tn, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
