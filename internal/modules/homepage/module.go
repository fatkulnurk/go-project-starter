package homepage

import (
	apppubsub "github.com/fatkulnurk/go-project-starter/internal/application/pubsub"
	appschedule "github.com/fatkulnurk/go-project-starter/internal/application/schedule"
	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage/adapter/api"
	pubsubadapter "github.com/fatkulnurk/go-project-starter/internal/modules/homepage/adapter/pubsub"
	scheduleadapter "github.com/fatkulnurk/go-project-starter/internal/modules/homepage/adapter/schedule"
	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage/adapter/web"
	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage/application"
	"github.com/fatkulnurk/go-project-starter/internal/platform/view"
	"github.com/go-chi/chi/v5"
)

// Settings carries the branding values (app name, base URLs, year) used when
// rendering the homepage page and the info response.
type Settings struct {
	AppName string
	BaseURL string
	// AssetsBaseURL is the absolute base URL of static assets (defaults to
	// BaseURL when empty).
	AssetsBaseURL string
	Year          int
}

// Dependencies are wired by the composition root. Currently only the branding
// settings are needed; Publisher is optional and used by the schedule adapter
// to broadcast the demo pub/sub events.
type Dependencies struct {
	Settings  Settings
	Publisher apppubsub.Publisher
}

// Module wires the homepage routes and their adapters. It exposes an empty API
// and registers both the HTML and JSON routes.
type Module struct {
	API       API
	settings  Settings
	publisher apppubsub.Publisher
}

// New constructs the homepage module from the supplied dependencies, storing
// the branding settings used by the registered routes.
func New(deps Dependencies) *Module {
	return &Module{
		API:       API{},
		settings:  deps.Settings,
		publisher: deps.Publisher,
	}
}

// RegisterWeb mounts the homepage HTML routes on the web router, supplying the
// branding common to the rendered page.
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

// RegisterAPI mounts the homepage JSON API on the API router, exposing the
// branding as JSON at the root.
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

// RegisterSchedule registers the module's periodic jobs on a scheduler. Today
// this is the demo "homepage.tick" job that logs the current time every minute
// and broadcasts a demo pub/sub event on the module's publisher (if any).
func (m *Module) RegisterSchedule(r appschedule.Registrar) {
	scheduleadapter.Register(r, m.publisher)
}

// RegisterPubSub registers the module's pub/sub topic handlers on a subscriber.
// Today this is the demo "app.demo.ping" handler that logs received events.
func (m *Module) RegisterPubSub(r apppubsub.Registrar) {
	pubsubadapter.Register(r)
}
