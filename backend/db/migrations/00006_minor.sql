-- +goose Up

-- Аудитын 3-р багц: жижиг засварууд (2026-08-20).

-- roles.implies: өөрийгөө заахгүй + FK (tenant доторх бодит role руу).
ALTER TABLE roles ADD CONSTRAINT roles_implies_not_self
    CHECK (implies IS NULL OR implies <> code);
ALTER TABLE roles ADD CONSTRAINT roles_implies_fk
    FOREIGN KEY (tenant_id, implies) REFERENCES roles (tenant_id, code)
    ON DELETE SET NULL (implies);

-- sessions: дуусах хугацааны индекс (purge + lookup).
CREATE INDEX idx_sessions_expires ON sessions (expires_at);

-- Хугацаа дууссан session-уудыг цэвэрлэх (serve цагийн зайцаар дуудна).
-- +goose StatementBegin
CREATE FUNCTION auth_sessions_purge() RETURNS integer
LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, public AS $$
DECLARE n integer;
BEGIN
  DELETE FROM sessions WHERE expires_at < now();
  GET DIAGNOSTICS n = ROW_COUNT;
  RETURN n;
END $$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION auth_sessions_purge() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_sessions_purge() TO nexus_app, nexus_admin;

-- jsonb баганад хэмжээний тааз (задгай өсөлтөөс сэргийлнэ).
ALTER TABLE audit_log ADD CONSTRAINT audit_details_size
    CHECK (pg_column_size(details) <= 16384);
ALTER TABLE permissions ADD CONSTRAINT permissions_default_roles_size
    CHECK (pg_column_size(default_roles) <= 2048);
ALTER TABLE platform_settings ADD CONSTRAINT platform_settings_value_size
    CHECK (pg_column_size(value) <= 16384);

-- audit: TRUNCATE-аас ч хамгаална (append-only бүрэн).
CREATE TRIGGER audit_no_truncate BEFORE TRUNCATE ON audit_log
    EXECUTE FUNCTION audit_block_mutation();

-- audit hash: ':'-ээр нийлүүлдэг preimage нь талбар дамнасан collision
-- үүсгэж болно ("a:b"+"c" == "a"+"b:c") — jsonb объект нь түлхүүртэй,
-- хоёрдмол утгагүй тул түүгээр солино. (Одоо байгаа гинжүүд хоосон үед
-- нэвтрүүлсэн — хуучин мөр рехэш хийх шаардлагагүй.)
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION audit_row_hash(
    p_prev char(64), p_tenant uuid, p_user uuid, p_action varchar(128),
    p_object varchar(255), p_details jsonb, p_at timestamptz)
RETURNS char(64)
LANGUAGE sql IMMUTABLE
SET search_path = pg_catalog, public AS $$
  SELECT encode(sha256(convert_to(jsonb_build_object(
    'prev', p_prev, 'tenant', p_tenant::text, 'user', coalesce(p_user::text, ''),
    'action', p_action, 'object', p_object, 'details', p_details,
    'at', to_char(p_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
    'UTF8')), 'hex')
$$;
-- +goose StatementEnd

-- Ирээдүйн (модулийн) объектод default эрхүүд — GRANT мартсанаас болж
-- runtime 42501 гарахаас сэргийлнэ. Миграцыг nexus_owner ажиллуулдаг тул
-- энэ нь owner-ийн үүсгэх бүх шинэ объектод үйлчилнэ.
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO nexus_app, nexus_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO nexus_app, nexus_admin;

-- +goose Down
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM nexus_app, nexus_admin;
DROP TRIGGER audit_no_truncate ON audit_log;
ALTER TABLE platform_settings DROP CONSTRAINT platform_settings_value_size;
ALTER TABLE permissions DROP CONSTRAINT permissions_default_roles_size;
ALTER TABLE audit_log DROP CONSTRAINT audit_details_size;
DROP FUNCTION auth_sessions_purge();
DROP INDEX idx_sessions_expires;
ALTER TABLE roles DROP CONSTRAINT roles_implies_fk;
ALTER TABLE roles DROP CONSTRAINT roles_implies_not_self;
