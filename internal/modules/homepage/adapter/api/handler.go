package api

import (
	"net/http"

	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
)

type handler struct {
	deps Deps
}

// info renders the branding as JSON.
func (h *handler) info(w http.ResponseWriter, _ *http.Request) {
	platformhttp.WriteSuccess(w, http.StatusOK, h.deps.Info)
}
