-- +goose Up

-- apps: store-ийн каталог. Бинари асахдаа компиллогдсон модулиуд +
-- registry/локал каталогоос (admin pool-оор) sync хийнэ. compiled нь энэ
-- бинарид байгаа эсэх — байхгүй апп store-д харагдавч "татаж авах" заавар
-- үзүүлнэ, суулгаж болохгүй.
CREATE TABLE apps (
    id          varchar(128) PRIMARY KEY,
    short_id    varchar(32) UNIQUE NOT NULL,
    name        varchar(160) NOT NULL,
    version     varchar(32) NOT NULL,
    description varchar(1000) NOT NULL DEFAULT '',
    publisher   varchar(120) NOT NULL DEFAULT '',
    -- Go модулийн зам — `nexus-mini add` CLI үүгээр татна (үе 2).
    go_module   varchar(255) NOT NULL DEFAULT '',
    compiled    boolean NOT NULL DEFAULT false,
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE app_installations (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    app_id       varchar(128) NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    version      varchar(32) NOT NULL,
    status       varchar(16) NOT NULL DEFAULT 'enabled' CHECK (status IN ('enabled', 'disabled')),
    installed_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, app_id)
);

ALTER TABLE apps              ENABLE ROW LEVEL SECURITY;
ALTER TABLE app_installations ENABLE ROW LEVEL SECURITY;

CREATE POLICY apps_select ON apps FOR SELECT USING (true);
CREATE POLICY apps_write ON apps FOR ALL
    USING (app_is_platform()) WITH CHECK (app_is_platform());

CREATE POLICY installations_all ON app_installations FOR ALL USING (
    tenant_id = app_tenant_id() OR app_is_platform()
) WITH CHECK (
    tenant_id = app_tenant_id() OR app_is_platform()
);

GRANT SELECT, INSERT, UPDATE, DELETE ON apps, app_installations TO nexus_app, nexus_admin;

-- +goose Down
DROP TABLE app_installations;
DROP TABLE apps;
