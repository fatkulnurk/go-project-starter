package api

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/command"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/go-chi/chi/v5"
)

type createRoleRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (h *handler) createRole(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.CreateRole.Execute(r.Context(), command.CreateRoleCommand{Code: req.Code, Name: req.Name}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusCreated, "role created")
}

func (h *handler) listRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.deps.ListRoles.Execute(r.Context())
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccess(w, http.StatusOK, toRoleResponses(roles))
}

func (h *handler) getRole(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	res, err := h.deps.GetRole.Execute(r.Context(), code)
	if err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccess(w, http.StatusOK, toRoleResponse(*res))
}

type updateRoleRequest struct {
	Name string `json:"name"`
}

func (h *handler) updateRole(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	var req updateRoleRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.UpdateRole.Execute(r.Context(), command.UpdateRoleCommand{Code: code, NewName: req.Name}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusOK, "role updated")
}

func (h *handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if err := h.deps.DeleteRole.Execute(r.Context(), command.DeleteRoleCommand{Code: code}); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	platformhttp.WriteSuccessMessage(w, http.StatusOK, "role deleted")
}
