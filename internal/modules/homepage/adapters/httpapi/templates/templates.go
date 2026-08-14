// Package templates renders the homepage module's pages from embedded Go
// templates. Pages are composed by attaching this module's content templates
// to the shared base layout owned by internal/platform/view; the content
// itself stays here so the platform only owns "the how".
//
// Templates are parsed once at startup via template.Must, so a syntax error
// fails the process immediately instead of surfacing on first render.
package templates

import (
	"embed"
	htmltemplate "html/template"

	"github.com/fatkulnurk/go-project-starter/internal/platform/view"
)

//go:embed *.html
var files embed.FS

// Common is the branding embedded by every view model. It aliases the view
// layout data so templates read {{.Common.AppName}} unchanged.
type Common = view.Common

// WelcomeData is the view model for welcome.html.
type WelcomeData struct {
	Common
}

// welcomeT composes the module's welcome content onto the platform base view
// layout once at startup.
var welcomeT = mustWelcome()

// mustWelcome parses the shared platform base view layout, then attaches the
// welcome content template from this module.
func mustWelcome() *htmltemplate.Template {
	layout, err := view.NewLayout()
	if err != nil {
		panic(err)
	}
	return htmltemplate.Must(layout.ParseFS(files, "welcome.html"))
}

// RenderWelcome executes the welcome template and returns the resulting HTML.
func RenderWelcome(data WelcomeData) (string, error) {
	return view.Render(welcomeT, "welcome", data)
}
