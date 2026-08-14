package http

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

// MountStatic serves the directory rooted at publicDir under the given base
// path (e.g. "/assets"). It is used by the API and web servers so email and
// web templates can reference {ASSETS_BASE_URL}{basePath}/<file>.
//
// http.FileServer already rejects path traversal, and the security headers
// middleware disables MIME sniffing so a stray HTML file cannot masquerade as
// a different content type.
func MountStatic(r chi.Router, publicDir, basePath string) {
	abs, err := filepath.Abs(publicDir)
	if err == nil {
		publicDir = abs
	}
	if _, err := os.Stat(publicDir); err != nil {
		// Directory may not exist yet in a fresh checkout; serve empty instead
		// of crashing the process.
		return
	}
	fs := http.FileServer(http.Dir(publicDir))
	r.Handle(basePath+"/*", http.StripPrefix(basePath, fs))
	r.Get(basePath, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, basePath+"/", http.StatusMovedPermanently)
	})
}
