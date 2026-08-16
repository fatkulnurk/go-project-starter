package http

import (
	"encoding/json"
	"errors"
	"io"
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
// data is marshaled under the "data" key; callers should pass structs or
// maps that serialize cleanly to JSON.
func WriteSuccess(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, dataEnvelope{Data: data})
}

// WriteSuccessMessage renders a success envelope carrying a plain message.
// The message is wrapped as {"data": {"message": ...}} via WriteSuccess.
func WriteSuccessMessage(w http.ResponseWriter, status int, message string) {
	WriteSuccess(w, status, map[string]string{"message": message})
}

// WriteError renders an error envelope with an explicit status and code.
// The payload is {"error": {"code", "message"}}; the caller picks the HTTP
// status and a stable machine-readable code.
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

// maxBodyBytes caps request bodies so unbounded payloads cannot exhaust memory.
const maxBodyBytes = 1 << 20 // 1 MiB

// DecodeJSON parses the request body into v, returning apierr.ErrInvalid on
// malformed input and apierr.ErrPayloadTooLarge when the body exceeds
// maxBodyBytes, so callers can map them with WriteMappedError. Trailing data
// after the single JSON value is rejected, and the body is drained so the
// connection can be reused.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return apierr.ErrPayloadTooLarge
		}
		return apierr.ErrInvalid
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return apierr.ErrInvalid
	}
	_, _ = io.Copy(io.Discard, r.Body)
	return nil
}
