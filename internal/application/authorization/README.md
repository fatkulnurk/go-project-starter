# `internal/application/authorization/`

**Cross-cutting capability** — the authorization contract: *is the caller
allowed to perform this action on this resource?* The RBAC module implements
`Authorizer`; protected routes receive it via middleware.

## Key types

| Symbol      | Purpose                                                    |
|-------------|------------------------------------------------------------|
| `Authorizer`| answers `HasPermission(identity, perm)` and `HasRole(identity, role)`; returns nil when allowed, `ErrForbidden` when not |
| `Identity`  | minimal caller description: `UserID` + `Roles` asserted at authentication time |

## Well-known constants

```go
RoleSuperAdmin       = "super_admin"
RoleUser             = "user"
PermissionManageRBAC = "rbac.manage"
```

RBAC seeds these and the auth module assigns the default role, so the constants
never drift from what is stored.

## Usage

```go
if err := authz.HasPermission(ctx, identity, authorization.PermissionManageRBAC); err != nil {
    return err // forbidden
}
```

`internal/platform/http.RequirePermission` / `RequireRole` wrap this contract as
route middleware (401 when unauthenticated, 403 when denied).

## Dependency rules

Vendor-free contract; imports stdlib only.
