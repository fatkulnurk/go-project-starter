package api

import (
	"net/http"

	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
)

// writeError maps any use-case error to the standardized error envelope and
// writes it to the response. It is the single error-rendering path of the API.
func writeError(w http.ResponseWriter, err error) {
	platformhttp.WriteMappedError(w, err)
}

// writeSuccess renders the standardized success envelope with the given status
// code and JSON payload.
func writeSuccess(w http.ResponseWriter, status int, data any) {
	platformhttp.WriteSuccess(w, status, data)
}

// decodeJSON parses the request body and, on failure, writes the mapped error
// to the response (malformed JSON maps to 422 invalid). It returns the same
// error so the caller can abort the handler.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	if err := platformhttp.DecodeJSON(w, r, v); err != nil {
		writeError(w, err)
		return err
	}
	return nil
}
