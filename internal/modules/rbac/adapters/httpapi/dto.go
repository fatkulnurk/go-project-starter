package httpapi

import (
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/domain"
)

type roleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type permissionResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type userAccessResponse struct {
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

func toRoleResponses(roles []*domain.Role) []roleResponse {
	out := make([]roleResponse, 0, len(roles))
	for _, r := range roles {
		out = append(out, roleResponse{ID: r.ID, Name: r.Name})
	}
	return out
}

func toPermissionResponses(perms []*domain.Permission) []permissionResponse {
	out := make([]permissionResponse, 0, len(perms))
	for _, p := range perms {
		out = append(out, permissionResponse{ID: p.ID, Name: p.Name})
	}
	return out
}
