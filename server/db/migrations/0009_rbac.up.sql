-- 0009_rbac — identity and authorisation for the admin surface.
--
-- Until now every write was guarded by one shared bearer token. That is fine
-- for a single operator and useless the moment two people need different
-- access, or anyone needs to know who changed a price.
--
-- The model is deny-by-default: a session grants exactly the permissions its
-- roles carry, and a route that declares no permission serves nobody.
--
-- The shared token does NOT go away -- it stays as a MACHINE credential for
-- automation, holding every permission and attributed in the audit log as
-- "service-token" rather than a person. Removing it would break the deploy
-- scripts and cron jobs documented in 15-backend-engine.md.

CREATE TABLE admin_permissions (
    code        TEXT PRIMARY KEY,
    label       TEXT NOT NULL,
    group_name  TEXT NOT NULL DEFAULT 'general',
    sort_order  INT  NOT NULL DEFAULT 0
);

CREATE TABLE admin_roles (
    id          UUID        PRIMARY KEY,
    code        TEXT        NOT NULL UNIQUE,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    -- A system role is seeded and cannot be deleted. Without this an admin can
    -- remove the only role that grants user.manage and lock everyone out of
    -- user administration permanently.
    is_system   BOOLEAN     NOT NULL DEFAULT FALSE,
    sort_order  INT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE admin_role_permissions (
    role_id         UUID NOT NULL REFERENCES admin_roles(id) ON DELETE CASCADE,
    permission_code TEXT NOT NULL REFERENCES admin_permissions(code) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_code)
);

CREATE TABLE admin_users (
    id            UUID        PRIMARY KEY,
    -- Stored lower-cased and trimmed by the application; the unique index is
    -- on the raw column, so normalising on the way in is what prevents
    -- "Ven@x.com" and "ven@x.com" becoming two accounts.
    email         TEXT        NOT NULL UNIQUE,
    name          TEXT        NOT NULL,
    -- argon2id, PHC format. Never a plaintext or reversible value.
    password_hash TEXT        NOT NULL,
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    -- Brute-force defence. failed_attempts resets on any successful login;
    -- locked_until is what the login path actually checks.
    failed_attempts INT       NOT NULL DEFAULT 0,
    locked_until  TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (email = lower(email)),
    CHECK (length(email) BETWEEN 3 AND 320),
    CHECK (length(name)  BETWEEN 1 AND 120)
);

CREATE TABLE admin_user_roles (
    user_id UUID NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES admin_roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- Sessions rather than JWTs: this is one service with one database, and a
-- session that can be revoked the instant someone leaves is worth more here
-- than a stateless token that stays valid until it expires.
CREATE TABLE admin_sessions (
    id           UUID        PRIMARY KEY,
    user_id      UUID        NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    -- The cookie carries the token; the database stores only its SHA-256. A
    -- leaked database backup therefore contains no usable session.
    token_hash   TEXT        NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_agent   TEXT        NOT NULL DEFAULT '',
    ip           TEXT        NOT NULL DEFAULT '',
    CHECK (expires_at > created_at)
);

CREATE INDEX ON admin_sessions (user_id);
CREATE INDEX ON admin_sessions (expires_at);

-- Every write is recorded. The point is answering "who changed this price and
-- when", which the shared token could never answer.
CREATE TABLE admin_audit_log (
    id         UUID        PRIMARY KEY,
    -- Null when the actor was the service token: there is no person to name.
    user_id    UUID        REFERENCES admin_users(id) ON DELETE SET NULL,
    actor      TEXT        NOT NULL,
    action     TEXT        NOT NULL,
    target     TEXT        NOT NULL DEFAULT '',
    detail     JSONB       NOT NULL DEFAULT '{}',
    ip         TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON admin_audit_log (created_at DESC);
CREATE INDEX ON admin_audit_log (user_id, created_at DESC);
CREATE INDEX ON admin_audit_log (action);

-- ── Permissions ──────────────────────────────────────────────────────────────
-- These codes are mirrored as Go constants in internal/platform/security.
-- TestPermissionConstantsMatchTheSeed fails if the two ever drift.
INSERT INTO admin_permissions (code, label, group_name, sort_order) VALUES
  ('menu.read',     'Lihat menu mingguan',        'menu',    10),
  ('menu.write',    'Ubah menu mingguan',         'menu',    20),
  ('menu.publish',  'Terbitkan / tarik menu',     'menu',    30),
  ('price.read',    'Lihat harga',                'price',   10),
  ('price.write',   'Ubah harga',                 'price',   20),
  ('rules.read',    'Lihat aturan harga',         'price',   30),
  ('rules.write',   'Ubah aturan harga',          'price',   40),
  ('content.read',  'Lihat konten & parameter',   'content', 10),
  ('content.write', 'Ubah konten & parameter',    'content', 20),
  ('user.manage',   'Kelola pengguna & peran',    'admin',   10),
  ('audit.read',    'Lihat log aktivitas',        'admin',   20);

-- ── Roles ────────────────────────────────────────────────────────────────────
INSERT INTO admin_roles (id, code, name, description, is_system, sort_order) VALUES
  ('00000000-0000-7000-8000-000000000001', 'owner',   'Owner',
   'Akses penuh, termasuk mengelola pengguna.', TRUE, 10),
  ('00000000-0000-7000-8000-000000000002', 'manager', 'Manager',
   'Semua kecuali mengelola pengguna.', TRUE, 20),
  ('00000000-0000-7000-8000-000000000003', 'editor',  'Editor Menu',
   'Ubah menu dan konten; harga hanya bisa dilihat.', TRUE, 30),
  ('00000000-0000-7000-8000-000000000004', 'viewer',  'Viewer',
   'Hanya melihat.', TRUE, 40);

-- owner: everything
INSERT INTO admin_role_permissions (role_id, permission_code)
SELECT '00000000-0000-7000-8000-000000000001', code FROM admin_permissions;

-- manager: everything except user.manage
INSERT INTO admin_role_permissions (role_id, permission_code)
SELECT '00000000-0000-7000-8000-000000000002', code
  FROM admin_permissions WHERE code <> 'user.manage';

-- editor: menu and content, plus read-only on price
INSERT INTO admin_role_permissions (role_id, permission_code)
SELECT '00000000-0000-7000-8000-000000000003', code
  FROM admin_permissions
 WHERE code IN ('menu.read','menu.write','menu.publish','content.read','content.write',
                'price.read','rules.read');

-- viewer: every read
INSERT INTO admin_role_permissions (role_id, permission_code)
SELECT '00000000-0000-7000-8000-000000000004', code
  FROM admin_permissions WHERE code LIKE '%.read';
