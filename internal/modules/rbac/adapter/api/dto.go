package api

import (
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/query"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

type roleResponse struct {
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type permissionResponse struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Group string `json:"group"`
	Name  string `json:"name"`
}

type userAccessResponse struct {
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

func toRoleResponse(r query.RoleDetail) roleResponse {
	return roleResponse{ID: r.ID, Code: r.Code, Name: r.Name, Permissions: r.Permissions}
}

func toRoleResponses(roles []query.RoleDetail) []roleResponse {
	out := make([]roleResponse, 0, len(roles))
	for _, r := range roles {
		out = append(out, toRoleResponse(r))
	}
	return out
}

func toPermissionResponses(perms []*domain.Permission) []permissionResponse {
	out := make([]permissionResponse, 0, len(perms))
	for _, p := range perms {
		out = append(out, permissionResponse{ID: p.ID, Code: p.Code, Group: p.Group, Name: p.Name})
	}
	return out
}
