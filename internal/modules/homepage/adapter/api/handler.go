package api

import (
	"net/http"

	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
)

type handler struct {
	deps Deps
}

// info renders the branding info as JSON via the shared success envelope.
func (h *handler) info(w http.ResponseWriter, _ *http.Request) {
	platformhttp.WriteSuccess(w, http.StatusOK, h.deps.Info)
}
