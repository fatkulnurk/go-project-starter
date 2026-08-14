package mailer

import (
	"embed"

	htmltemplate "html/template"
)

//go:embed layout.html
var layoutFS embed.FS

// Common carries branding shared by the email layout. Module email templates
// embed it so the layout can render the brand header and footer.
type Common struct {
	AppName string
	BaseURL string
	Year    int
}

// NewEmailLayout parses the shared email layout into a template set that
// defines the "layout" shell and a "content" block. Modules attach their own
// content templates to it with (*html/template.Template).ParseFS.
func NewEmailLayout() (*htmltemplate.Template, error) {
	return htmltemplate.ParseFS(layoutFS, "layout.html")
}
