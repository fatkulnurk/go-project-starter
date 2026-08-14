-- =============================================================================
-- RBAC (Role-Based Access Control): 5 tabel yang saling berhubungan.
--   roles            -> daftar peran (mis. 'admin', 'user')
--   permissions      -> daftar izin aksi (mis. 'media.upload', 'rbac.manage')
--   role_permissions -> izin yang dimiliki sebuah peran (many-to-many)
--   user_roles       -> peran yang dimiliki seorang user (many-to-many)
--   user_permissions -> izin tambahan langsung ke user (override/luar peran)
-- Otorisasi pada sebuah request = izin via peran user + izin langsung user.
-- =============================================================================

-- =============================================================================
-- TABLE: roles
-- Kumpulan peran yang bisa disematkan ke user. Peran default ('admin', 'user')
-- dibuat saat bootstrap aplikasi. Semua id berisi UUID v7 (36 char) dari app.
--
-- Contoh data:
--   id         = '0195c5d7-5c2f-7d00-8000-000000000001'  (UUID v7)
--   name       = 'admin' | 'user'         (unik)
--   created_at = '2026-01-15 10:30:00'
--   updated_at = '2026-01-15 10:30:00'
-- =============================================================================
CREATE TABLE IF NOT EXISTS roles (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (dibuat di app)
    name       VARCHAR(64)  NOT NULL,             -- nama peran (unik)
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (name)
);

-- =============================================================================
-- TABLE: permissions
-- Daftar izin granular yang bisa dicek di endpoint. Satu izin = satu aksi API.
--
-- Contoh data:
--   id         = '0195c5d7-6d30-7d00-8000-000000000001'  (UUID v7)
--   name       = 'media.upload' | 'rbac.manage' | 'media.delete'   (unik)
--   created_at = '2026-01-15 10:30:00'
--   updated_at = '2026-01-15 10:30:00'
-- =============================================================================
CREATE TABLE IF NOT EXISTS permissions (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY, -- UUID v7 (dibuat di app)
    name       VARCHAR(64)  NOT NULL,             -- nama izin (unik)
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (name)
);

-- =============================================================================
-- TABLE: role_permissions
-- Menghubungkan peran dengan izin (many-to-many). Izin yang dimiliki semua
-- user berperan 'user' harus diisi di sini agar tidak ditambahkan satu-satu.
-- created_at/updated_at diisi otomatis oleh database.
--
-- Contoh data:
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
-- Menghubungkan user dengan peran (many-to-many). Satu user bisa punya banyak
-- peran; izinnya digabung dari semua peran miliknya.
--
-- Contoh data:
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
-- Izin khusus yang diberikan langsung ke user, terlepas dari perannya.
-- Dipakai untuk pengecualian: beri izin ke satu user tanpa membuat peran baru.
--
-- Contoh data:
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
