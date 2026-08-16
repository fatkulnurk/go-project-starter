# `internal/modules/rbac/`

**Business module** — the role-based access control module: roles, permissions,
role↔user and permission↔user links, plus the authorizer used by protected
endpoints. Modeled after Spatie permissions: `super_admin` implicitly holds
every permission.

## What it does

- Roles and permissions CRUD (by stable `code`), role↔permission sync
- Assign/revoke roles and direct permissions for a user
- Effective permission resolution (direct grants + role inheritance) with a
  versioned permission cache
- `Authorizer` — implements `application/authorization.Authorizer`, used by
  `platform/http.RequirePermission`
- `Bootstrap` — idempotently ensures the well-known roles/permissions at startup

## How it works

```text
HTTP /api/v1/rbac/*            (adapter/api — requires auth + rbac.manage)
        │  parse request → one use case → standardized response
        ▼
application/command|query      (use cases: CRUD, assignments, bootstrap)
        │  domain entities + repositories; every change bumps the cache version
        ▼
domain/ + infrastructure/      (pure rules + database/sql repositories)
```

Admin routes in `adapter/api` are guarded by `Authenticate` +
`RequirePermission(authz, "rbac.manage")`; every route calls exactly one use
case.

## Layers

| Layer            | Contents                                                            |
|------------------|---------------------------------------------------------------------|
| `domain/`        | `Role`, `Permission` + `RoleRepository`, `PermissionRepository`, `UserAccessRepository` interfaces |
| `application/command/` | one struct per use case (`CreateRole`, `AssignRole`, `GrantPermission`, `SyncRolePermissions`, `Bootstrap`, ...); `CacheBumper` for invalidation; protected-guards (last `super_admin`, immutable built-ins) |
| `application/query/` | `ListRoles`, `ListPermissions`, `GetRole`, `GetUser` (roles + effective permissions) |
| `application/cache/` | `PermissionCache` — versioned user-role/permission caching           |
| `infrastructure/`  | `database/sql` repositories (`?` placeholders, rebound for postgres) |
| `adapter/api/`     | routes under `/api/v1/rbac` (roles, permissions, user assignments)  |
| `seeder/`          | `DefaultRoles`/`DefaultPermissions` + `RolesPermissionsSeeder` (registered as `rbac.roles_permissions`) |

## Permission cache

`application/cache.PermissionCache` stores each user's roles/permissions in the
shared cache (redis/memory) tagged with a global version counter (`rbac:ver`).
Any RBAC change **bumps** the version, which invalidates every user entry at
once (no per-user scans). Stale entries are simply ignored; changes propagate
within `RBAC_CACHE_TTL`. A nil cache disables caching entirely.

## Public surface (`module.go`)

- `New(deps Dependencies) *Module` — wires repositories, the optional permission
  cache, every use case, and the `Service`/`Authorizer`/`API` surfaces. It never
  fails.
- `Service()` (`api.go`) — the **narrow public interface other modules depend
  on** (`RolesAndPermissions`, `HasPermission`, `HasRole`, `AssignRole`). The
  auth module consumes it for role lookup and the default `user` role.
- `Authorizer()` — the `application/authorization.Authorizer` implementation for
  protected routes (empty permission/role = allowed; super_admin bypasses all
  checks).
- `API` — every use case, exposed to `RegisterAPI`.
- `Bootstrap(ctx, opts)` — idempotent; the composition root runs it after wiring
  so a fresh database is usable immediately (registration relies on the default
  `user` role existing).
- `RegisterAPI(r, authn, authz)` — mounts the admin routes behind auth +
  `rbac.manage`.
