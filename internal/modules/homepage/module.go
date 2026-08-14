package homepage

import (
	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage/adapters/api"
	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage/adapters/web"
	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage/application"
	"github.com/fatkulnurk/go-project-starter/internal/platform/view"
	"github.com/go-chi/chi/v5"
)

// Settings carries branding for the rendered page.
type Settings struct {
	AppName string
	BaseURL string
	// AssetsBaseURL is the absolute base URL of static assets (defaults to
	// BaseURL when empty).
	AssetsBaseURL string
	Year          int
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

// RegisterWeb mounts the homepage HTML routes on the web router.
func (m *Module) RegisterWeb(r chi.Router) {
	web.RegisterRoutes(r, web.Deps{
		Common: view.Common{
			AppName:       m.settings.AppName,
			BaseURL:       m.settings.BaseURL,
			AssetsBaseURL: m.settings.AssetsBaseURL,
			Year:          m.settings.Year,
		},
	})
}

// RegisterAPI mounts the homepage JSON API on the API router.
func (m *Module) RegisterAPI(r chi.Router) {
	api.RegisterRoutes(r, api.Deps{
		Info: application.Info{
			AppName:       m.settings.AppName,
			BaseURL:       m.settings.BaseURL,
			AssetsBaseURL: m.settings.AssetsBaseURL,
			Year:          m.settings.Year,
		},
	})
}
