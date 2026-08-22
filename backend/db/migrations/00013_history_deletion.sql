-- +goose Up

-- OGN-ээс авсан 2 зүйл (2026-08-23): аппын хувилбарын түүх + байгууллага
-- устгалын 30 хоногийн хүлээлт.

-- Нийтлэгчийн гаргасан хувилбарууд (компиллогдсон/registry-ээс харагдсан).
CREATE TABLE app_releases (
    app_id   varchar(128) NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    version  varchar(32)  NOT NULL,
    seen_at  timestamptz  NOT NULL DEFAULT now(),
    PRIMARY KEY (app_id, version)
);
ALTER TABLE app_releases ENABLE ROW LEVEL SECURITY;
CREATE POLICY app_releases_select ON app_releases FOR SELECT USING (true);
CREATE POLICY app_releases_platform ON app_releases FOR ALL USING (app_is_platform()) WITH CHECK (app_is_platform());
GRANT SELECT ON app_releases TO nexus_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON app_releases TO nexus_admin;

-- Энэ tenant юу хийсэн: суулгах/асаах/унтраах/шинэчлэх.
CREATE TABLE installation_events (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    app_id       varchar(128) NOT NULL,
    action       varchar(16) NOT NULL CHECK (action IN ('install', 'enable', 'disable', 'upgrade')),
    from_version varchar(32) NOT NULL DEFAULT '',
    to_version   varchar(32) NOT NULL DEFAULT '',
    user_id      uuid REFERENCES users(id) ON DELETE SET NULL,
    at           timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_installation_events ON installation_events(tenant_id, app_id, id DESC);
ALTER TABLE installation_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY installation_events_tenant ON installation_events FOR ALL
    USING (tenant_id = app_tenant_id() OR app_is_platform())
    WITH CHECK (tenant_id = app_tenant_id() OR app_is_platform());
GRANT SELECT, INSERT ON installation_events TO nexus_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON installation_events TO nexus_admin;

-- Устгалын хүлээлт: товлосон цаг; өнгөрсний дараа цагийн sweep устгана.
ALTER TABLE tenants ADD COLUMN deletion_scheduled_at timestamptz;
-- tenants-д DELETE policy байгаагүй (апп role хэзээ ч устгахгүй) — зөвхөн
-- платформ (nexus_admin) sweep-д.
CREATE POLICY tenants_delete ON tenants FOR DELETE USING (app_is_platform());

DROP FUNCTION tenant_state(uuid);
-- +goose StatementBegin
CREATE FUNCTION tenant_state(p_tenant uuid)
RETURNS TABLE (suspended boolean, reason varchar(300), read_only boolean, deletion_at timestamptz)
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public, pg_temp AS $$
  SELECT t.suspended_at IS NOT NULL, t.suspension_reason, t.read_only, t.deletion_scheduled_at
    FROM tenants t WHERE t.id = p_tenant
$$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION tenant_state(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION tenant_state(uuid) TO nexus_auth;

-- +goose Down
DROP FUNCTION tenant_state(uuid);
-- +goose StatementBegin
CREATE FUNCTION tenant_state(p_tenant uuid)
RETURNS TABLE (suspended boolean, reason varchar(300), read_only boolean)
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public, pg_temp AS $$
  SELECT t.suspended_at IS NOT NULL, t.suspension_reason, t.read_only FROM tenants t WHERE t.id = p_tenant
$$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION tenant_state(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION tenant_state(uuid) TO nexus_auth;
DROP POLICY tenants_delete ON tenants;
ALTER TABLE tenants DROP COLUMN deletion_scheduled_at;
DROP TABLE installation_events;
DROP TABLE app_releases;
