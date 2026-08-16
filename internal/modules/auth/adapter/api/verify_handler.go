package api

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/command"
)

type verifyEmailRequest struct {
	Email   string `json:"email"`
	Code    string `json:"code"`
	OldCode string `json:"old_code"`
}

func (h *handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if err := h.deps.VerifyEmail.Execute(r.Context(), command.VerifyEmailCommand{Email: req.Email, Code: req.Code, OldCode: req.OldCode}); err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{responseVerified: true})
}

type verifyPhoneRequest struct {
	Phone   string `json:"phone"`
	Code    string `json:"code"`
	OldCode string `json:"old_code"`
}

func (h *handler) verifyPhone(w http.ResponseWriter, r *http.Request) {
	var req verifyPhoneRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if err := h.deps.VerifyPhone.Execute(r.Context(), command.VerifyPhoneCommand{Phone: req.Phone, Code: req.Code, OldCode: req.OldCode}); err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{responseVerified: true})
}
