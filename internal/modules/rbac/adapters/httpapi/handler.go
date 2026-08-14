package httpapi

import (
	"net/http"

	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/commands"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/go-chi/chi/v5"
)

type handler struct {
	deps Deps
}

type createRoleRequest struct {
	Name string `json:"name"`
}

func (h *handler) createRole(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.CreateRole.Execute(r.Context(), commands.CreateRoleCommand{Name: req.Name}); err != nil {
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

type createPermissionRequest struct {
	Name string `json:"name"`
}

func (h *handler) createPermission(w http.ResponseWriter, r *http.Request) {
	var req createPermissionRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.CreatePermission.Execute(r.Context(), commands.CreatePermissionCommand{Name: req.Name}); err != nil {
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

type syncRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

func (h *handler) syncRolePermissions(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var req syncRolePermissionsRequest
	if err := platformhttp.DecodeJSON(w, r, &req); err != nil {
		platformhttp.WriteMappedError(w, err)
		return
	}
	if err := h.deps.SyncRolePermissions.Execute(r.Context(), commands.SyncRolePermissionsCommand{
		Role: name, Permissions: req.Permissions,
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
	if err := h.deps.AssignRole.Execute(r.Context(), commands.AssignRoleCommand{UserID: userID, Role: req.Role}); err != nil {
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
	if err := h.deps.RevokeRole.Execute(r.Context(), commands.RevokeRoleCommand{UserID: userID, Role: req.Role}); err != nil {
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
	if err := h.deps.GrantPermission.Execute(r.Context(), commands.GrantPermissionCommand{UserID: userID, Permission: req.Permission}); err != nil {
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
	if err := h.deps.RevokePermission.Execute(r.Context(), commands.RevokePermissionCommand{UserID: userID, Permission: req.Permission}); err != nil {
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
