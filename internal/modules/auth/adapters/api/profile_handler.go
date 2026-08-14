package api

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/commands"
)

type updateProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func (h *handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	id, err := identity(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	var req updateProfileRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	res, err := h.deps.UpdateProfile.Execute(r.Context(), commands.UpdateProfileCommand{
		UserID: id.UserID,
		Name:   req.Name,
		Email:  req.Email,
		Phone:  req.Phone,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	body := map[string]any{responseUser: toUserResponse(res.User)}
	if res.DevEmailCode != "" {
		body[responseDevEmailCode] = res.DevEmailCode
	}
	if res.DevPhoneCode != "" {
		body[responseDevPhoneCode] = res.DevPhoneCode
	}
	writeSuccess(w, http.StatusOK, body)
}
