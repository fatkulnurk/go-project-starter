package httpapi

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/commands"
)

type forgotPasswordRequest struct {
	Identifier string `json:"identifier"`
}

func (h *handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	res, err := h.deps.ForgotPassword.Execute(r.Context(), commands.ForgotPasswordCommand{
		Identifier: req.Identifier,
		IP:         clientIP(r),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	body := map[string]any{responseExpiresIn: int64(res.ExpiresIn.Seconds())}
	writeSuccess(w, http.StatusOK, body)
}

type resetPasswordRequest struct {
	Identifier  string `json:"identifier"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

func (h *handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if err := h.deps.ResetPassword.Execute(r.Context(), commands.ResetPasswordCommand{
		Identifier: req.Identifier, Code: req.Code, NewPassword: req.NewPassword,
	}); err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{responseReset: true})
}
