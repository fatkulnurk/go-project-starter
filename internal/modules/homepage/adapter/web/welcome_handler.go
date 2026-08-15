package web

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage/template"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
)

// welcome renders the landing page for GET /.
func (h *handler) welcome(w http.ResponseWriter, _ *http.Request) {
	html, err := template.RenderWelcome(template.WelcomeData{Common: h.deps.Common})
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}
