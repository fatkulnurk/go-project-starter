package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/application/apierr"
	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
)

// Standardized API envelope. Every success is {"data": ...}, every failure is
// {"error": {"code", "message"}}. Modules must use these helpers instead of
// writing raw JSON so every endpoint behaves identically.

type dataEnvelope struct {
	Data any `json:"data"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteSuccess renders a success envelope with the given status.
func WriteSuccess(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, dataEnvelope{Data: data})
}

// WriteSuccessMessage renders a success envelope carrying a plain message.
func WriteSuccessMessage(w http.ResponseWriter, status int, message string) {
	WriteSuccess(w, status, map[string]string{"message": message})
}

// WriteError renders an error envelope with an explicit status and code.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorEnvelope{Error: errorBody{Code: code, Message: message}})
}

// WriteMappedError renders an error envelope derived from err's kind. It knows
// the cross-cutting application sentinels and honors apierr.StatusCoder /
// apierr.ErrorCoder implemented by domain errors. Unknown errors become a 500
// "internal" response and never leak their message.
func WriteMappedError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, appauth.ErrUnauthenticated):
		WriteError(w, http.StatusUnauthorized, "unauthenticated", err.Error())
	case errors.Is(err, authorization.ErrForbidden):
		WriteError(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		kind := apierr.KindOf(err)
		msg := err.Error()
		if kind.Code == apierr.KindInternal.Code {
			msg = "internal error"
		}
		WriteError(w, kind.Status, kind.Code, msg)
	}
}

// WriteJSON renders a raw JSON payload (used internally and by the server
// bootstrap). Prefer WriteSuccess/WriteError in module handlers.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return
	}
}

// DecodeJSON parses the request body into v, returning apierr.ErrInvalid on
// malformed input so callers can map it with WriteMappedError.
func DecodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return apierr.ErrInvalid
	}
	return nil
}
