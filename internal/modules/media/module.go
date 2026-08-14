package media

import (
	"database/sql"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
	"github.com/fatkulnurk/go-project-starter/internal/application/storage"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/adapters/httpapi"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/application/queries"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/infrastructure"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
	"github.com/go-chi/chi/v5"
)

// Dependencies are wired by the composition root.
type Dependencies struct {
	DB       *sql.DB
	DBDriver string
	Storage  storage.Storage
	// URLGenerator produces public URLs for stored objects. When nil (or a
	// driver without public serving, e.g. local), media URLs fall back to the
	// API download endpoint built from BaseURL.
	URLGenerator  storage.URLGenerator
	BaseURL       string
	Disk          string
	Auditor       audit.Auditor
	MaxUploadSize int64
	// Location is the app timezone (nil means UTC).
	Location *time.Location
}

// Module wires the media use cases and their adapters.
type Module struct {
	API           API
	maxUploadSize int64
	urlGenerator  storage.URLGenerator
	baseURL       string
}

// New constructs the media module.
func New(deps Dependencies) *Module {
	repo := infrastructure.NewMediaRepository(deps.DB, deps.DBDriver)
	return &Module{
		API: API{
			AddMedia:    commands.NewAddMedia(repo, deps.Storage, deps.Disk, deps.Auditor, clock.Real{Loc: deps.Location}),
			RemoveMedia: commands.NewRemoveMedia(repo, deps.Storage, deps.Auditor),
			GetMedia:    queries.NewGetMedia(repo, deps.Storage),
			ListByModel: queries.NewListByModel(repo),
		},
		maxUploadSize: deps.MaxUploadSize,
		urlGenerator:  deps.URLGenerator,
		baseURL:       deps.BaseURL,
	}
}

// RegisterHTTP mounts the media routes. Reads require authentication; writes
// additionally require the media.manage permission.
func (m *Module) RegisterHTTP(r chi.Router, authn appauth.Authenticator, authz authorization.Authorizer) {
	httpapi.RegisterRoutes(r, httpapi.Deps{
		AddMedia:      m.API.AddMedia,
		RemoveMedia:   m.API.RemoveMedia,
		GetMedia:      m.API.GetMedia,
		ListByModel:   m.API.ListByModel,
		Authenticator: authn,
		Authorizer:    authz,
		MaxUploadSize: m.maxUploadSize,
		URLGenerator:  m.urlGenerator,
		BaseURL:       m.baseURL,
	})
}
