package api

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/command"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/go-chi/chi/v5"
)

type syncRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

func (h *handler) syncRolePermissions(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	var req syncRolePermissionsRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.SyncRolePermissions.Execute(r.Context(), command.SyncRolePermissionsCommand{
		Role: code, Permissions: req.Permissions,
	}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusOK, "role permissions synced")
}

type assignRoleRequest struct {
	Role string `json:"role"`
}

func (h *handler) assignRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req assignRoleRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.AssignRole.Execute(r.Context(), command.AssignRoleCommand{UserID: userID, Role: req.Role}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusOK, "role assigned")
}

func (h *handler) revokeRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req assignRoleRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.RevokeRole.Execute(r.Context(), command.RevokeRoleCommand{UserID: userID, Role: req.Role}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusOK, "role revoked")
}

type grantPermissionRequest struct {
	Permission string `json:"permission"`
}

func (h *handler) grantPermission(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req grantPermissionRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.GrantPermission.Execute(r.Context(), command.GrantPermissionCommand{UserID: userID, Permission: req.Permission}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusOK, "permission granted")
}

func (h *handler) revokePermission(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req grantPermissionRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.RevokePermission.Execute(r.Context(), command.RevokePermissionCommand{UserID: userID, Permission: req.Permission}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusOK, "permission revoked")
}

func (h *handler) getUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	res, err := h.deps.GetUser.Execute(r.Context(), userID)
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccess(w, http.StatusOK, userAccessResponse{
		Roles:       res.Roles,
		Permissions: res.Permissions,
	})
}
