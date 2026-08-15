package api

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/command"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/go-chi/chi/v5"
)

type createPermissionRequest struct {
	Code  string `json:"code"`
	Group string `json:"group"`
	Name  string `json:"name"`
}

func (h *handler) createPermission(w http.ResponseWriter, r *http.Request) {
	var req createPermissionRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.CreatePermission.Execute(r.Context(), command.CreatePermissionCommand{Code: req.Code, Group: req.Group, Name: req.Name}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusCreated, "permission created")
}

func (h *handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.deps.ListPermissions.Execute(r.Context())
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccess(w, http.StatusOK, toPermissionResponses(perms))
}

type updatePermissionRequest struct {
	Group string `json:"group"`
	Name  string `json:"name"`
}

func (h *handler) updatePermission(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	var req updatePermissionRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.UpdatePermission.Execute(r.Context(), command.UpdatePermissionCommand{Code: code, NewGroup: req.Group, NewName: req.Name}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusOK, "permission updated")
}

func (h *handler) deletePermission(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if err := h.deps.DeletePermission.Execute(r.Context(), command.DeletePermissionCommand{Code: code}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusOK, "permission deleted")
}
