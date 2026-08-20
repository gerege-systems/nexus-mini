-- +goose Up

CREATE TABLE devices (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       varchar(120) NOT NULL,
    kind       varchar(64) NOT NULL DEFAULT '',
    serial     varchar(120) NOT NULL,
    status     varchar(16) NOT NULL DEFAULT 'active'
               CHECK (status IN ('active', 'repair', 'lost', 'retired')),
    note       varchar(500) NOT NULL DEFAULT '',
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, serial)
);
CREATE INDEX idx_devices_tenant ON devices(tenant_id, created_at DESC);

ALTER TABLE devices ENABLE ROW LEVEL SECURITY;
CREATE POLICY devices_tenant ON devices FOR ALL USING (
    tenant_id = app_tenant_id() OR app_is_platform()
) WITH CHECK (
    tenant_id = app_tenant_id() OR app_is_platform()
);

GRANT SELECT, INSERT, UPDATE, DELETE ON devices TO nexus_app, nexus_admin;

-- +goose Down
DROP TABLE devices;
