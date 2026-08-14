-- =============================================================================
-- RBAC (Role-Based Access Control): 5 related tables.
--   roles            -> list of roles (e.g. 'admin', 'user')
--   permissions      -> list of action permissions (e.g. 'media.upload', 'rbac.manage')
--   role_permissions -> permissions held by a role (many-to-many)
--   user_roles       -> roles held by a user (many-to-many)
--   user_permissions -> extra permissions granted directly to a user (override/outside roles)
-- Authorization for a request = permissions via the user's roles + direct user permissions.
-- =============================================================================

-- =============================================================================
-- TABLE: roles
-- Roles that can be assigned to users. The default roles ('admin', 'user')
-- are created during application bootstrap. All ids are app-generated UUID v7
-- (36 chars).
--
-- Example data:
--   id         = '0195c5d7-5c2f-7d00-8000-000000000001'  (UUID v7)
--   name       = 'admin' | 'user'         (unique)
--   created_at = '2026-01-15 10:30:00'
--   updated_at = '2026-01-15 10:30:00'
-- =============================================================================
CREATE TABLE IF NOT EXISTS roles (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (generated in app)
    name       VARCHAR(64)  NOT NULL,             -- role name (unique)
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (name)
);

-- =============================================================================
-- TABLE: permissions
-- Granular permissions checked at endpoints. One permission = one API action.
--
-- Example data:
--   id         = '0195c5d7-6d30-7d00-8000-000000000001'  (UUID v7)
--   name       = 'media.upload' | 'rbac.manage' | 'media.delete'   (unique)
--   created_at = '2026-01-15 10:30:00'
--   updated_at = '2026-01-15 10:30:00'
-- =============================================================================
CREATE TABLE IF NOT EXISTS permissions (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (generated in app)
    name       VARCHAR(64)  NOT NULL,             -- permission name (unique)
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (name)
);

-- =============================================================================
-- TABLE: role_permissions
-- Links roles to permissions (many-to-many). Permissions shared by every user
-- with the 'user' role must be populated here so they are not added one by one.
-- created_at/updated_at are filled automatically by the database.
--
-- Example data:
--   role_id       = '0195c5d7-5c2f-7d00-8000-000000000001' (admin)
--   permission_id = '0195c5d7-6d30-7d00-8000-000000000001' (media.upload)
-- =============================================================================
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       VARCHAR(36) NOT NULL, -- FK roles.id
    permission_id VARCHAR(36) NOT NULL, -- FK permissions.id
    created_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_rp_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
    CONSTRAINT fk_rp_permission FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
);

-- =============================================================================
-- TABLE: user_roles
-- Links users to roles (many-to-many). A user can hold many roles; their
-- permissions are the union of all their roles.
--
-- Example data:
--   user_id    = '0195c5d4-2b40-7d00-8000-000000000001' (Budi)
--   role_id    = '0195c5d7-5c2f-7d00-8000-000000000001' (admin)
-- =============================================================================
CREATE TABLE IF NOT EXISTS user_roles (
    user_id    VARCHAR(36) NOT NULL, -- FK users.id
    role_id    VARCHAR(36) NOT NULL, -- FK roles.id
    created_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id),
    CONSTRAINT fk_ur_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_ur_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE
);

-- =============================================================================
-- TABLE: user_permissions
-- Permissions granted directly to a user, regardless of their roles.
-- Used for exceptions: grant one user a permission without creating a new role.
--
-- Example data:
--   user_id       = '0195c5d4-2b40-7d00-8000-000000000001' (Budi)
--   permission_id = '0195c5d7-7e31-7d00-8000-000000000001' (media.delete)
-- =============================================================================
CREATE TABLE IF NOT EXISTS user_permissions (
    user_id       VARCHAR(36) NOT NULL, -- FK users.id
    permission_id VARCHAR(36) NOT NULL, -- FK permissions.id
    created_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, permission_id),
    CONSTRAINT fk_up_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_up_permission FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
);

CREATE INDEX idx_user_roles_user ON user_roles (user_id);
CREATE INDEX idx_user_permissions_user ON user_permissions (user_id);
