package api

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/command"
)

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

func (h *handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	res, err := h.deps.Register.Execute(r.Context(), command.RegisterCommand{
		Name: req.Name, Email: req.Email, Phone: req.Phone, Password: req.Password,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	body := map[string]any{responseUserID: res.UserID}
	if res.DevEmailCode != "" {
		body[responseDevEmailCode] = res.DevEmailCode
	}
	if res.DevPhoneCode != "" {
		body[responseDevPhoneCode] = res.DevPhoneCode
	}
	writeSuccess(w, http.StatusCreated, body)
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	res, err := h.deps.Login.Execute(r.Context(), command.LoginCommand{
		Identifier: req.Identifier,
		Password:   req.Password,
		IP:         clientIP(r),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, toTokenResponse(res.AccessToken, res.RefreshToken, res.ExpiresIn, res.User))
}
