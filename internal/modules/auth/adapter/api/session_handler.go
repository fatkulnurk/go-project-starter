package api

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/command"
)

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	res, err := h.deps.Refresh.Execute(r.Context(), command.RefreshCommand{RefreshToken: req.RefreshToken})
	if err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{
		responseAccessToken:  res.AccessToken,
		responseRefreshToken: res.RefreshToken,
		responseTokenType:    tokenTypeBearer,
		responseExpiresIn:    int64(res.ExpiresIn.Seconds()),
	})
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	id, err := identity(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	var req logoutRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if err := h.deps.Logout.Execute(r.Context(), command.LogoutCommand{
		UserID:       id.UserID,
		RefreshToken: req.RefreshToken,
	}); err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{responseLoggedOut: true})
}

func (h *handler) me(w http.ResponseWriter, r *http.Request) {
	id, err := identity(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	profile, err := h.deps.Profile.Execute(r.Context(), id.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, toProfileResponse(profile))
}
