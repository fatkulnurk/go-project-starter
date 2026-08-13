package httpapi

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
)

// writeError maps any use-case error to the standardized error envelope.
func writeError(w http.ResponseWriter, err error) {
	platformhttp.WriteMappedError(w, err)
}

// writeSuccess renders the standardized success envelope.
func writeSuccess(w http.ResponseWriter, status int, data any) {
	platformhttp.WriteSuccess(w, status, data)
}

// decodeJSON parses the request body; malformed JSON maps to 422 invalid.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	if err := platformhttp.DecodeJSON(r, v); err != nil {
		writeError(w, domain.ErrInvalid)
		return err
	}
	return nil
}
