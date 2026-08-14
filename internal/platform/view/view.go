// Package view renders HTML pages (browser views) from embedded Go templates.
// It owns the shared base layout — the "how" of every page — while business
// modules attach their own content templates via ParseFS, mirroring the email
// layout in internal/platform/mailer.
package view

import (
	"embed"

	htmltemplate "html/template"

	"github.com/fatkulnurk/go-project-starter/internal/application/branding"
)

//go:embed base.html
var baseFS embed.FS

// Common carries branding shared by the base layout. Module page templates
// embed it so the layout can render the header and footer.
type Common = branding.Common

// NewLayout parses the shared base layout into a template set that defines
// the "layout" shell and a "content" block. Modules attach their own content
// templates to it with (*html/template.Template).ParseFS.
func NewLayout() (*htmltemplate.Template, error) {
	return htmltemplate.ParseFS(baseFS, "base.html")
}
