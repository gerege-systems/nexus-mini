-- +goose Up

-- Үе 3 (2026-08-23): OIDC provider. Хүснэгтүүд зөвхөн nexus_auth role-д
-- (токен/код/түлхүүр — апп/модулийн SQL-д хэзээ ч харагдахгүй); клиентийн
-- бүртгэл tenant-ийнх (portal UI апп pool-оор), nexus_auth уншина.

CREATE TABLE oidc_keys (
    kid         varchar(32) PRIMARY KEY,
    private_pem varchar(4000) NOT NULL,
    public_pem  varchar(1000) NOT NULL,
    active      boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now()
);
REVOKE ALL ON oidc_keys FROM nexus_app, nexus_admin;
GRANT SELECT, INSERT, UPDATE ON oidc_keys TO nexus_auth;

CREATE TABLE oauth_clients (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    client_id          varchar(64) UNIQUE NOT NULL,
    client_secret_hash varchar(255),               -- NULL = public client (PKCE only)
    name               varchar(120) NOT NULL,
    redirect_uris      jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (pg_column_size(redirect_uris) <= 4096),
    post_logout_uris   jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (pg_column_size(post_logout_uris) <= 4096),
    scopes             varchar(255) NOT NULL DEFAULT 'openid profile email',
    created_by         uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_oauth_clients_tenant ON oauth_clients(tenant_id);
ALTER TABLE oauth_clients ENABLE ROW LEVEL SECURITY;
CREATE POLICY oauth_clients_tenant ON oauth_clients FOR ALL
    USING (tenant_id = app_tenant_id() OR app_is_platform() OR current_user = 'nexus_auth')
    WITH CHECK (tenant_id = app_tenant_id() OR app_is_platform());
GRANT SELECT, INSERT, UPDATE, DELETE ON oauth_clients TO nexus_app, nexus_admin;
GRANT SELECT ON oauth_clients TO nexus_auth;

-- Authorization code (PKCE заавал), нэг удаа.
CREATE TABLE oauth_codes (
    code_hash      char(64) PRIMARY KEY,
    client_id      varchar(64) NOT NULL,
    tenant_id      uuid NOT NULL,
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redirect_uri   varchar(1000) NOT NULL,
    scope          varchar(255) NOT NULL,
    nonce          varchar(255) NOT NULL DEFAULT '',
    code_challenge varchar(128) NOT NULL,
    expires_at     timestamptz NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now()
);
REVOKE ALL ON oauth_codes FROM nexus_app, nexus_admin;
GRANT SELECT, INSERT, DELETE ON oauth_codes TO nexus_auth;

-- Access (opaque) + refresh токен. family — refresh rotation-ий гэр бүл:
-- хүчингүй refresh дахин ирвэл (replay) бүх гэр бүлийг хүчингүй болгоно.
CREATE TABLE oauth_tokens (
    token_hash  char(64) PRIMARY KEY,
    kind        varchar(8) NOT NULL CHECK (kind IN ('access', 'refresh')),
    family      uuid NOT NULL,
    client_id   varchar(64) NOT NULL,
    tenant_id   uuid NOT NULL,
    user_id     uuid REFERENCES users(id) ON DELETE CASCADE,  -- NULL = client_credentials
    scope       varchar(255) NOT NULL,
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_oauth_tokens_family ON oauth_tokens(family);
CREATE INDEX idx_oauth_tokens_exp ON oauth_tokens(expires_at);
REVOKE ALL ON oauth_tokens FROM nexus_app, nexus_admin;
GRANT SELECT, INSERT, UPDATE, DELETE ON oauth_tokens TO nexus_auth;

-- Зөвшөөрөл санах: нэг хэрэглэгч нэг клиентэд нэг удаа.
CREATE TABLE oauth_consents (
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id  varchar(64) NOT NULL,
    scope      varchar(255) NOT NULL,
    granted_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, client_id)
);
REVOKE ALL ON oauth_consents FROM nexus_app, nexus_admin;
GRANT SELECT, INSERT, UPDATE, DELETE ON oauth_consents TO nexus_auth;

-- Клиент устгахад түүний токен/зөвшөөрөл хамт (FK биш — өөр role-ийн хүснэгт).
-- +goose StatementBegin
CREATE FUNCTION oauth_client_cleanup() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
BEGIN
  DELETE FROM oauth_tokens WHERE client_id = OLD.client_id;
  DELETE FROM oauth_codes WHERE client_id = OLD.client_id;
  DELETE FROM oauth_consents WHERE client_id = OLD.client_id;
  RETURN OLD;
END $$;
-- +goose StatementEnd
CREATE TRIGGER oauth_clients_cleanup AFTER DELETE ON oauth_clients
  FOR EACH ROW EXECUTE FUNCTION oauth_client_cleanup();

-- Хугацаа дууссан код/токенуудыг цэвэрлэх (цагийн ticker, nexus_auth).
-- +goose StatementBegin
CREATE FUNCTION oauth_purge() RETURNS integer
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
DECLARE n integer; m integer;
BEGIN
  DELETE FROM oauth_codes WHERE expires_at < clock_timestamp();
  GET DIAGNOSTICS n = ROW_COUNT;
  DELETE FROM oauth_tokens WHERE expires_at < clock_timestamp() - interval '1 day';
  GET DIAGNOSTICS m = ROW_COUNT;
  RETURN n + m;
END $$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION oauth_purge() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION oauth_purge() TO nexus_auth;

-- nexus_auth-д хэрэгтэй уншилтууд (RLS хүснэгтэд шууд эрхгүй тул definer).
-- +goose StatementBegin
CREATE FUNCTION auth_is_member(p_tenant uuid, p_user uuid) RETURNS boolean
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM memberships m WHERE m.tenant_id = p_tenant AND m.user_id = p_user)
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION tenant_public_name(p_tenant uuid) RETURNS TABLE (name varchar(160))
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public, pg_temp AS $$
  SELECT t.name FROM tenants t WHERE t.id = p_tenant
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION oidc_user_claims(p_user uuid, p_tenant uuid)
RETURNS TABLE (name varchar(120), email varchar(255), tenant_slug varchar(64), roles varchar(64)[])
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public, pg_temp AS $$
  SELECT u.name, u.email, t.slug,
         coalesce((SELECT array_agg(r.code ORDER BY r.code)
                     FROM memberships m JOIN membership_roles mr ON mr.membership_id = m.id
                     JOIN roles r ON r.id = mr.role_id
                    WHERE m.tenant_id = p_tenant AND m.user_id = p_user), '{}')
    FROM users u, tenants t WHERE u.id = p_user AND t.id = p_tenant
$$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION auth_is_member(uuid, uuid), tenant_public_name(uuid), oidc_user_claims(uuid, uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_is_member(uuid, uuid), tenant_public_name(uuid), oidc_user_claims(uuid, uuid) TO nexus_auth;

-- +goose Down
DROP FUNCTION oidc_user_claims(uuid, uuid);
DROP FUNCTION tenant_public_name(uuid);
DROP FUNCTION auth_is_member(uuid, uuid);
DROP FUNCTION oauth_purge();
DROP TRIGGER oauth_clients_cleanup ON oauth_clients;
DROP FUNCTION oauth_client_cleanup();
DROP TABLE oauth_consents;
DROP TABLE oauth_tokens;
DROP TABLE oauth_codes;
DROP TABLE oauth_clients;
DROP TABLE oidc_keys;
