-- +goose Up

-- ЗАРЧИМ (docs/01-lessons.md):
--   * Апп зөвхөн nexus_app role-оор холбогдоно — RLS түүнд бүрэн үйлчилнэ.
--   * Миграцыг nexus_owner ажиллуулна; SECURITY DEFINER функцууд owner-ийн
--     эрхээр гүйж RLS-ийг тойрдог тул FORCE тавьдаггүй (owner-оор апп
--     хэзээ ч холбогдохгүй).
--   * Нэвтрэлт/танилттай холбоотой бүх хайлт (session, имэйл, setup төлөв)
--     tenant тодорхойлогдохоос өмнө хийгддэг тул ЭХНЭЭСЭЭ definer функцээр.
--   * Платформын админ эрхийг GUC биш DB role-оор таньдаг (pg_has_role).

-- +goose StatementBegin
CREATE FUNCTION app_tenant_id() RETURNS uuid
LANGUAGE sql STABLE AS $$
  SELECT NULLIF(current_setting('app.tenant_id', true), '')::uuid
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app_user_id() RETURNS uuid
LANGUAGE sql STABLE AS $$
  SELECT NULLIF(current_setting('app.user_id', true), '')::uuid
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION app_is_platform() RETURNS boolean
LANGUAGE sql STABLE AS $$
  SELECT pg_has_role(current_user, 'nexus_platform', 'member')
$$;
-- +goose StatementEnd

CREATE TABLE users (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email          varchar(255) UNIQUE NOT NULL,
    password_hash  varchar(255) NOT NULL,
    name           varchar(120) NOT NULL,
    platform_admin boolean NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tenants (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       varchar(64) UNIQUE NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,62}$'),
    name       varchar(160) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE memberships (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, user_id)
);
CREATE INDEX idx_memberships_user ON memberships(user_id);

CREATE TABLE roles (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code      varchar(64) NOT NULL CHECK (code ~ '^[a-z][a-z0-9_]{1,63}$'),
    name      varchar(120) NOT NULL,
    -- implies: энэ role нь доод role-ийн эрхийг өвлөнө (admin ⊃ manager ⊃ user).
    -- Нэг эцэгтэй жижиг гинж — Odoo-гийн implied_ids-ийн mini хувилбар.
    implies   varchar(64),
    active    boolean NOT NULL DEFAULT true,
    UNIQUE (tenant_id, code)
);

-- permissions: глобал каталог. Бинари асахдаа компиллогдсон модулиудын
-- тунхаглалаас (admin pool-оор) sync хийдэг.
CREATE TABLE permissions (
    code          varchar(128) PRIMARY KEY,
    module_id     varchar(128) NOT NULL,
    name          varchar(160) NOT NULL,
    description   varchar(500) NOT NULL DEFAULT '',
    own_scope     boolean NOT NULL DEFAULT false,
    default_roles jsonb NOT NULL DEFAULT '[]'
);

CREATE TABLE role_permissions (
    role_id         uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_code varchar(128) NOT NULL REFERENCES permissions(code) ON DELETE CASCADE,
    scope           varchar(3) NOT NULL DEFAULT 'all' CHECK (scope IN ('all', 'own')),
    PRIMARY KEY (role_id, permission_code)
);

CREATE TABLE membership_roles (
    membership_id uuid NOT NULL REFERENCES memberships(id) ON DELETE CASCADE,
    role_id       uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (membership_id, role_id)
);

CREATE TABLE sessions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Идэвхтэй tenant. NULL = хараахан сонгоогүй (шинэ хэрэглэгч).
    tenant_id  uuid REFERENCES tenants(id) ON DELETE SET NULL,
    token_hash char(64) UNIQUE NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_user ON sessions(user_id);

CREATE TABLE platform_settings (
    key   varchar(64) PRIMARY KEY,
    value jsonb NOT NULL
);

-- ─── RLS ────────────────────────────────────────────────────────────────

ALTER TABLE users             ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants           ENABLE ROW LEVEL SECURITY;
ALTER TABLE memberships       ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles             ENABLE ROW LEVEL SECURITY;
ALTER TABLE permissions       ENABLE ROW LEVEL SECURITY;
ALTER TABLE role_permissions  ENABLE ROW LEVEL SECURITY;
ALTER TABLE membership_roles  ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions          ENABLE ROW LEVEL SECURITY;
ALTER TABLE platform_settings ENABLE ROW LEVEL SECURITY;

-- users: өөрийгөө + өөрийн tenant-ийн гишүүдийг харна; зөвхөн өөрийгөө засна.
CREATE POLICY users_select ON users FOR SELECT USING (
    id = app_user_id()
    OR EXISTS (SELECT 1 FROM memberships m
               WHERE m.user_id = users.id AND m.tenant_id = app_tenant_id())
    OR app_is_platform()
);
CREATE POLICY users_update ON users FOR UPDATE
    USING (id = app_user_id() OR app_is_platform());

-- tenants: идэвхтэй tenant + өөрийн гишүүнчлэлтэй tenant-ууд (tenant
-- сонгохоос ӨМНӨ жагсаалт харагдах ёстой — өмнөх төслийн сургамж #1).
CREATE POLICY tenants_select ON tenants FOR SELECT USING (
    id = app_tenant_id()
    OR EXISTS (SELECT 1 FROM memberships m
               WHERE m.tenant_id = tenants.id AND m.user_id = app_user_id())
    OR app_is_platform()
);
-- Бүртгэлтэй хэрэглэгч бүр өөрийн байгууллагаа үүсгэж болно (self-signup).
CREATE POLICY tenants_insert ON tenants FOR INSERT
    WITH CHECK (app_user_id() IS NOT NULL);
CREATE POLICY tenants_update ON tenants FOR UPDATE
    USING (id = app_tenant_id() OR app_is_platform());

CREATE POLICY memberships_select ON memberships FOR SELECT USING (
    tenant_id = app_tenant_id() OR user_id = app_user_id() OR app_is_platform()
);
CREATE POLICY memberships_write ON memberships FOR ALL USING (
    tenant_id = app_tenant_id() OR app_is_platform()
) WITH CHECK (
    tenant_id = app_tenant_id() OR app_is_platform()
);

CREATE POLICY roles_all ON roles FOR ALL USING (
    tenant_id = app_tenant_id() OR app_is_platform()
) WITH CHECK (
    tenant_id = app_tenant_id() OR app_is_platform()
);

-- permissions: каталог бүх нэвтэрсэн хэрэглэгчид харагдана; бичих нь
-- зөвхөн платформ (boot sync).
CREATE POLICY permissions_select ON permissions FOR SELECT USING (true);
CREATE POLICY permissions_write ON permissions FOR ALL
    USING (app_is_platform()) WITH CHECK (app_is_platform());

CREATE POLICY role_permissions_all ON role_permissions FOR ALL USING (
    EXISTS (SELECT 1 FROM roles r
            WHERE r.id = role_permissions.role_id
              AND (r.tenant_id = app_tenant_id() OR app_is_platform()))
) WITH CHECK (
    EXISTS (SELECT 1 FROM roles r
            WHERE r.id = role_permissions.role_id
              AND (r.tenant_id = app_tenant_id() OR app_is_platform()))
);

CREATE POLICY membership_roles_all ON membership_roles FOR ALL USING (
    EXISTS (SELECT 1 FROM memberships m
            WHERE m.id = membership_roles.membership_id
              AND (m.tenant_id = app_tenant_id() OR app_is_platform()))
) WITH CHECK (
    EXISTS (SELECT 1 FROM memberships m
            WHERE m.id = membership_roles.membership_id
              AND (m.tenant_id = app_tenant_id() OR app_is_platform()))
);

-- sessions: хэрэглэгч зөвхөн өөрийн session-уудыг харж/устгана.
-- Үүсгэх/token-оор хайх нь доорх definer функцуудаар л явна.
CREATE POLICY sessions_own ON sessions FOR SELECT USING (user_id = app_user_id());
CREATE POLICY sessions_delete ON sessions FOR DELETE USING (user_id = app_user_id());

CREATE POLICY platform_settings_select ON platform_settings FOR SELECT USING (true);
CREATE POLICY platform_settings_write ON platform_settings FOR ALL
    USING (app_is_platform()) WITH CHECK (app_is_platform());

-- ─── Pre-auth давхарга: SECURITY DEFINER функцууд ──────────────────────
-- Эдгээр нь nexus_owner-ийн эрхээр гүйж RLS-ийг тойрно. Зөвхөн
-- шаардлагатай баганыг буцаана, өөр юу ч ил гаргахгүй.

-- +goose StatementBegin
CREATE FUNCTION auth_user_by_email(p_email varchar(255))
RETURNS TABLE (id uuid, password_hash varchar(255), name varchar(120), platform_admin boolean)
LANGUAGE sql SECURITY DEFINER STABLE AS $$
  SELECT u.id, u.password_hash, u.name, u.platform_admin
    FROM users u WHERE u.email = lower(p_email)
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION auth_signup(p_email varchar(255), p_password_hash varchar(255), p_name varchar(120))
RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE v_id uuid;
BEGIN
  INSERT INTO users (email, password_hash, name)
  VALUES (lower(p_email), p_password_hash, p_name)
  RETURNING users.id INTO v_id;
  RETURN v_id;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION auth_session_create(p_user_id uuid, p_token_hash char(64), p_expires_at timestamptz)
RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE v_id uuid;
BEGIN
  INSERT INTO sessions (user_id, token_hash, expires_at)
  VALUES (p_user_id, p_token_hash, p_expires_at)
  RETURNING sessions.id INTO v_id;
  RETURN v_id;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION auth_session_lookup(p_token_hash char(64))
RETURNS TABLE (session_id uuid, user_id uuid, tenant_id uuid, platform_admin boolean, name varchar(120), email varchar(255))
LANGUAGE sql SECURITY DEFINER STABLE AS $$
  SELECT s.id, s.user_id, s.tenant_id, u.platform_admin, u.name, u.email
    FROM sessions s JOIN users u ON u.id = s.user_id
   WHERE s.token_hash = p_token_hash AND s.expires_at > now()
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION auth_session_set_tenant(p_session_id uuid, p_tenant_id uuid)
RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE v_user uuid;
BEGIN
  SELECT s.user_id INTO v_user FROM sessions s WHERE s.id = p_session_id;
  IF v_user IS NULL THEN RETURN false; END IF;
  IF p_tenant_id IS NOT NULL AND NOT EXISTS (
       SELECT 1 FROM memberships m
        WHERE m.tenant_id = p_tenant_id AND m.user_id = v_user) THEN
    RETURN false;
  END IF;
  UPDATE sessions SET tenant_id = p_tenant_id WHERE id = p_session_id;
  RETURN true;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION auth_session_delete(p_token_hash char(64))
RETURNS void
LANGUAGE sql SECURITY DEFINER AS $$
  DELETE FROM sessions WHERE token_hash = p_token_hash
$$;
-- +goose StatementEnd

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO nexus_app, nexus_admin;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO nexus_app, nexus_admin;

-- +goose Down
DROP FUNCTION auth_session_delete(char);
DROP FUNCTION auth_session_set_tenant(uuid, uuid);
DROP FUNCTION auth_session_lookup(char);
DROP FUNCTION auth_session_create(uuid, char, timestamptz);
DROP FUNCTION auth_signup(varchar, varchar, varchar);
DROP FUNCTION auth_user_by_email(varchar);
DROP TABLE platform_settings;
DROP TABLE sessions;
DROP TABLE membership_roles;
DROP TABLE role_permissions;
DROP TABLE permissions;
DROP TABLE roles;
DROP TABLE memberships;
DROP TABLE tenants;
DROP TABLE users;
DROP FUNCTION app_is_platform();
DROP FUNCTION app_user_id();
DROP FUNCTION app_tenant_id();
