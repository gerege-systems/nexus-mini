-- +goose Up

-- Хэлтэс/нэгжийн мод + ажилтны байршил (OGN-ийн organisation аппаас авсан
-- санаа, nexus-mini-д модуль хэлбэрээр).
CREATE TABLE org_departments (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code        varchar(32)  NOT NULL,
    name        varchar(120) NOT NULL,
    parent_id   uuid REFERENCES org_departments(id) ON DELETE SET NULL,
    manager_membership_id uuid REFERENCES memberships(id) ON DELETE SET NULL,
    active      boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, code),
    CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE INDEX idx_org_departments_tenant ON org_departments(tenant_id, parent_id);

-- Гишүүнчлэл бүрт нэг байршил: хэлтэс + албан тушаал.
CREATE TABLE org_positions (
    membership_id uuid PRIMARY KEY REFERENCES memberships(id) ON DELETE CASCADE,
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    department_id uuid REFERENCES org_departments(id) ON DELETE SET NULL,
    job_title     varchar(120) NOT NULL DEFAULT '',
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_org_positions_dept ON org_positions(tenant_id, department_id);

ALTER TABLE org_departments ENABLE ROW LEVEL SECURITY;
ALTER TABLE org_positions   ENABLE ROW LEVEL SECURITY;
CREATE POLICY org_departments_tenant ON org_departments FOR ALL
    USING (tenant_id = app_tenant_id() OR app_is_platform())
    WITH CHECK (tenant_id = app_tenant_id() OR app_is_platform());
CREATE POLICY org_positions_tenant ON org_positions FOR ALL
    USING (tenant_id = app_tenant_id() OR app_is_platform())
    WITH CHECK (tenant_id = app_tenant_id() OR app_is_platform());
GRANT SELECT, INSERT, UPDATE, DELETE ON org_departments, org_positions TO nexus_app, nexus_admin;

-- +goose Down
DROP TABLE org_positions;
DROP TABLE org_departments;
