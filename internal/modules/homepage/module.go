package homepage

import (
	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage/adapters/httpapi"
	"github.com/fatkulnurk/go-project-starter/internal/platform/view"
	"github.com/go-chi/chi/v5"
)

// Settings carries branding for the rendered page.
type Settings struct {
	AppName string
	BaseURL string
	Year    int
}

// Dependencies are wired by the composition root.
type Dependencies struct {
	Settings Settings
}

// Module wires the homepage routes and their adapters.
type Module struct {
	API      API
	settings Settings
}

// New constructs the homepage module.
func New(deps Dependencies) *Module {
	return &Module{
		API:      API{},
		settings: deps.Settings,
	}
}

// RegisterHTTP mounts the homepage routes on the shared router.
func (m *Module) RegisterHTTP(r chi.Router) {
	httpapi.RegisterRoutes(r, httpapi.Deps{
		Common: view.Common{
			AppName: m.settings.AppName,
			BaseURL: m.settings.BaseURL,
			Year:    m.settings.Year,
		},
	})
}
