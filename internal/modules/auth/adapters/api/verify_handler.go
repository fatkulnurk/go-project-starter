package api

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/commands"
)

type verifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (h *handler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if err := h.deps.VerifyEmail.Execute(r.Context(), commands.VerifyEmailCommand{Email: req.Email, Code: req.Code}); err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{responseVerified: true})
}

type verifyPhoneRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

func (h *handler) verifyPhone(w http.ResponseWriter, r *http.Request) {
	var req verifyPhoneRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if err := h.deps.VerifyPhone.Execute(r.Context(), commands.VerifyPhoneCommand{Phone: req.Phone, Code: req.Code}); err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{responseVerified: true})
}
