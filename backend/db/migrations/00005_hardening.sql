-- +goose Up

-- Аудитын 2-р багц (2026-08-20).

-- audit_append: tenant-ийг дуудагчийн үгээр биш session-ий RLS context-оор
-- баталгаажуулна — модуль өөр tenant-ийн гинжид мөр шахаж чадахгүй.
-- (Definer дотор current_user нь owner тул холбогдсон жинхэнэ role-ийг
-- session_user-ээр шалгана.)
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION audit_append(
    p_tenant uuid, p_user uuid, p_action varchar(128),
    p_object varchar(255), p_details jsonb)
RETURNS bigint
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  v_prev char(64);
  v_at   timestamptz;
  v_id   bigint;
BEGIN
  IF p_tenant IS DISTINCT FROM app_tenant_id()
     AND NOT pg_has_role(session_user, 'nexus_platform', 'member') THEN
    RAISE EXCEPTION 'audit: tenant context зөрүүтэй';
  END IF;
  PERFORM pg_advisory_xact_lock(hashtext('audit:' || p_tenant::text));
  SELECT a.hash INTO v_prev FROM audit_log a
   WHERE a.tenant_id = p_tenant ORDER BY a.id DESC LIMIT 1;
  IF v_prev IS NULL THEN
    v_prev := repeat('0', 64);
  END IF;
  v_at := clock_timestamp();
  INSERT INTO audit_log (tenant_id, user_id, action, object, details, prev_hash, hash, occurred_at)
  VALUES (p_tenant, p_user, p_action, p_object, coalesce(p_details, '{}'::jsonb),
          v_prev,
          audit_row_hash(v_prev, p_tenant, p_user, p_action, p_object,
                         coalesce(p_details, '{}'::jsonb), v_at),
          v_at)
  RETURNING id INTO v_id;
  RETURN v_id;
END $$;
-- +goose StatementEnd

-- SECURITY DEFINER функцуудад search_path pin (Postgres-ийн албан зөвлөмж)
-- + PUBLIC-аас EXECUTE хураана (nexus_app/nexus_admin-ий grant хэвээр).
ALTER FUNCTION auth_user_by_email(varchar) SET search_path = pg_catalog, public;
ALTER FUNCTION auth_signup(varchar, varchar, varchar) SET search_path = pg_catalog, public;
ALTER FUNCTION auth_session_create(uuid, char, timestamptz) SET search_path = pg_catalog, public;
ALTER FUNCTION auth_session_lookup(char) SET search_path = pg_catalog, public;
ALTER FUNCTION auth_session_set_tenant(uuid, uuid) SET search_path = pg_catalog, public;
ALTER FUNCTION auth_session_delete(char) SET search_path = pg_catalog, public;
ALTER FUNCTION audit_append(uuid, uuid, varchar, varchar, jsonb) SET search_path = pg_catalog, public;
ALTER FUNCTION audit_verify(uuid) SET search_path = pg_catalog, public;

REVOKE EXECUTE ON FUNCTION auth_user_by_email(varchar) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION auth_signup(varchar, varchar, varchar) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION auth_session_create(uuid, char, timestamptz) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION auth_session_lookup(char) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION auth_session_set_tenant(uuid, uuid) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION auth_session_delete(char) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION audit_append(uuid, uuid, varchar, varchar, jsonb) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION audit_verify(uuid) FROM PUBLIC;

-- +goose Down
-- (audit_append-ийн хуучин хувилбарыг сэргээхгүй — down нь зөвхөн хоосон)
SELECT 1;
