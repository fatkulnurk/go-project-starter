package api

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/command"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/query"
	"github.com/go-chi/chi/v5"
)

// --- MFA (second step of a TOTP-protected login) ---------------------------

type verifyMFARequest struct {
	Challenge string `json:"challenge"`
	Code      string `json:"code"`
}

func (h *handler) verifyMFA(w http.ResponseWriter, r *http.Request) {
	var req verifyMFARequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	res, err := h.deps.VerifyMFA.Execute(r.Context(), command.VerifyMFACommand{
		Challenge: req.Challenge,
		Code:      req.Code,
		IP:        clientIP(r),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, toTokenResponse(res.AccessToken, res.RefreshToken, res.ExpiresIn, res.User))
}

// --- MFA setup/confirm/disable (authenticated) -----------------------------

func (h *handler) setupTOTP(w http.ResponseWriter, r *http.Request) {
	id, err := identity(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	res, err := h.deps.SetupTOTP.Execute(r.Context(), command.SetupTOTPCommand{UserID: id.UserID})
	if err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{
		responseSecret:       res.Secret,
		responseProvisioning: res.URI,
	})
}

type confirmTOTPRequest struct {
	Code string `json:"code"`
}

func (h *handler) confirmTOTP(w http.ResponseWriter, r *http.Request) {
	id, err := identity(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	var req confirmTOTPRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	res, err := h.deps.ConfirmTOTP.Execute(r.Context(), command.ConfirmTOTPCommand{UserID: id.UserID, Code: req.Code})
	if err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{
		responseTOTPEnabled: true,
		responseRecovery:    res.RecoveryCodes,
	})
}

type disableTOTPRequest struct {
	Code string `json:"code"`
}

func (h *handler) disableTOTP(w http.ResponseWriter, r *http.Request) {
	id, err := identity(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	var req disableTOTPRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if err := h.deps.DisableTOTP.Execute(r.Context(), command.DisableTOTPCommand{UserID: id.UserID, Code: req.Code}); err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{responseTOTPEnabled: false})
}

// --- Change password --------------------------------------------------------

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *handler) changePassword(w http.ResponseWriter, r *http.Request) {
	id, err := identity(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	var req changePasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if err := h.deps.ChangePassword.Execute(r.Context(), command.ChangePasswordCommand{
		UserID:      id.UserID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}); err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{responseReset: true})
}

// --- Sessions ---------------------------------------------------------------

func (h *handler) sessions(w http.ResponseWriter, r *http.Request) {
	id, err := identity(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	sessions, err := h.deps.Sessions.Execute(r.Context(), id.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, toSessionsResponse(sessions))
}

func (h *handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	id, err := identity(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.deps.SessionRevoke.Execute(r.Context(), command.SessionRevokeCommand{
		UserID:   id.UserID,
		FamilyID: chi.URLParam(r, "familyID"),
	}); err != nil {
		writeError(w, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{responseLoggedOut: true})
}

func toSessionsResponse(sessions []query.Session) map[string]any {
	if sessions == nil {
		sessions = []query.Session{}
	}
	return map[string]any{responseSessions: sessions}
}
