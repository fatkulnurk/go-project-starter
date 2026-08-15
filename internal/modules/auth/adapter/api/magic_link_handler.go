package api

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/command"
)

type magicLinkRequest struct {
	Email string `json:"email"`
}

func (h *handler) magicLinkRequest(w http.ResponseWriter, r *http.Request) {
	var req magicLinkRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	res, err := h.deps.MagicLinkRequest.Execute(r.Context(), command.MagicLinkRequestCommand{
		Email: req.Email,
		IP:    clientIP(r),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	body := map[string]any{responseExpiresIn: int64(res.ExpiresIn.Seconds())}
	writeSuccess(w, http.StatusOK, body)
}

type magicLinkVerifyRequest struct {
	Token string `json:"token"`
}

func (h *handler) magicLinkVerify(w http.ResponseWriter, r *http.Request) {
	var req magicLinkVerifyRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	res, err := h.deps.MagicLinkVerify.Execute(r.Context(), command.MagicLinkVerifyCommand{Token: req.Token})
	if err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, toTokenResponse(res.AccessToken, res.RefreshToken, res.ExpiresIn, res.User))
}

// magicLinkVerifyGet verifies a magic link delivered by email: the token
// travels in the query string so a plain click resolves the login.
func (h *handler) magicLinkVerifyGet(w http.ResponseWriter, r *http.Request) {
	res, err := h.deps.MagicLinkVerify.Execute(r.Context(), command.MagicLinkVerifyCommand{Token: r.URL.Query().Get("token")})
	if err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, toTokenResponse(res.AccessToken, res.RefreshToken, res.ExpiresIn, res.User))
}
