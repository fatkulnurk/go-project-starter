package media

import (
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/application/commands"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media/application/queries"
)

// API exposes the module's use cases.
type API struct {
	AddMedia    *commands.AddMedia
	RemoveMedia *commands.RemoveMedia
	GetMedia    *queries.GetMedia
	ListByModel *queries.ListByModel
}
