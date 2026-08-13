package httpapi

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/commands"
)

type magicLinkRequest struct {
	Email string `json:"email"`
}

func (h *handler) magicLinkRequest(w http.ResponseWriter, r *http.Request) {
	var req magicLinkRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	res, err := h.deps.MagicLinkRequest.Execute(r.Context(), commands.MagicLinkRequestCommand{
		Email: req.Email,
		IP:    clientIP(r),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	body := map[string]any{responseExpiresIn: int64(res.ExpiresIn.Seconds())}
	if res.DevLink != "" {
		body[responseDevLink] = res.DevLink
	}
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
	res, err := h.deps.MagicLinkVerify.Execute(r.Context(), commands.MagicLinkVerifyCommand{Token: req.Token})
	if err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, toTokenResponse(res.AccessToken, res.RefreshToken, res.ExpiresIn, res.User))
}
