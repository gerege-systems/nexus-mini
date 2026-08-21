-- +goose Up

-- OGN-ээс авсан 2 зүйл (2026-08-22): байгууллагын профайл + түдгэлзүүлэх /
-- зөвхөн-унших төлөв.

CREATE TABLE tenant_profiles (
    tenant_id           uuid PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    legal_name          varchar(200) NOT NULL DEFAULT '',
    registration_number varchar(32)  NOT NULL DEFAULT '',
    tax_number          varchar(32)  NOT NULL DEFAULT '',
    address             varchar(500) NOT NULL DEFAULT '',
    phone               varchar(32)  NOT NULL DEFAULT '',
    email               varchar(255) NOT NULL DEFAULT '',
    website             varchar(255) NOT NULL DEFAULT '',
    updated_at          timestamptz  NOT NULL DEFAULT now()
);
ALTER TABLE tenant_profiles ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_profiles_all ON tenant_profiles FOR ALL
    USING (tenant_id = app_tenant_id() OR app_is_platform())
    WITH CHECK (tenant_id = app_tenant_id() OR app_is_platform());
GRANT SELECT, INSERT, UPDATE, DELETE ON tenant_profiles TO nexus_app, nexus_admin;

ALTER TABLE tenants
    ADD COLUMN suspended_at      timestamptz,
    ADD COLUMN suspension_reason varchar(300) NOT NULL DEFAULT '',
    ADD COLUMN read_only         boolean NOT NULL DEFAULT false;

-- Апп role tenants-ийг зөвхөн name баганаар шинэчилнэ — түдгэлзүүлэлт,
-- read_only нь платформ админы (nexus_admin) л эрх. Модуль SDK-аар DB
-- хүрдэг тул багана түвшинд хаана.
REVOKE UPDATE ON tenants FROM nexus_app;
GRANT UPDATE (name) ON tenants TO nexus_app;

-- Төлөвийг хүсэлт бүрд (RequireTenant) уншина — identity GUC тавигдахаас
-- өмнө тул RLS-ээс чөлөөтэй definer функц (го-pg дүрэм #1).
-- +goose StatementBegin
CREATE FUNCTION tenant_state(p_tenant uuid)
RETURNS TABLE (suspended boolean, reason varchar(300), read_only boolean)
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public AS $$
  SELECT t.suspended_at IS NOT NULL, t.suspension_reason, t.read_only
    FROM tenants t WHERE t.id = p_tenant
$$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION tenant_state(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION tenant_state(uuid) TO nexus_app, nexus_admin;

-- +goose Down
DROP FUNCTION tenant_state(uuid);
GRANT UPDATE ON tenants TO nexus_app;
ALTER TABLE tenants DROP COLUMN suspended_at, DROP COLUMN suspension_reason, DROP COLUMN read_only;
DROP TABLE tenant_profiles;
