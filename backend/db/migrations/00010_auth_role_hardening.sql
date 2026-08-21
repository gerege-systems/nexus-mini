-- +goose Up

-- Аудит 2026-08-22 (DB давхарга):
--  H1  search_path-д pg_temp ХАМГИЙН СҮҮЛД байх ёстой — үгүй бол temp хүснэгт
--      definer функцийн дотоод relation-ийг дарна. Мөн nexus_app/nexus_admin-д
--      TEMPORARY эрхийг авна.
--  H2  pre-auth definer функцууд (session үүсгэх, signup, email-ээр хайх,
--      handover) nexus_app-аас дуудагдаж байсан → модулийн SQL/SQLi-ээр дурын
--      хэрэглэгчийн session үүсгэж болно. Тусдаа nexus_auth role (зөвхөн auth
--      pool) л дуудна. nexus_auth-г deploy/01-roles.sql үүсгэнэ.
--  M1  ALTER DEFAULT PRIVILEGES ... GRANT EXECUTE нь fail-open — авна.
--  M3  users.password_hash tenant-ийн гишүүдэд SELECT-ээр уншигдаж байсан.
--  M2  tenants INSERT-д suspended_at/read_only багана апп role-д хаалттай.

-- H1: pg_temp сүүлд
ALTER FUNCTION auth_user_by_email(varchar)                        SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION auth_signup(varchar, varchar, varchar)              SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION auth_session_create(uuid, char, timestamptz)        SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION auth_session_lookup(char)                           SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION auth_session_set_tenant(uuid, uuid)                 SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION auth_session_delete(char)                           SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION auth_sessions_purge()                               SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION auth_handover_create(char, uuid, uuid, uuid, timestamptz) SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION auth_handover_consume(char, char, timestamptz)      SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION audit_append(uuid, uuid, varchar, varchar, jsonb)   SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION audit_verify(uuid)                                  SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION tenant_state(uuid)                                  SET search_path = pg_catalog, public, pg_temp;
ALTER FUNCTION membership_role_same_tenant()                       SET search_path = pg_catalog, public, pg_temp;

-- +goose StatementBegin
DO $$
BEGIN
  EXECUTE format('REVOKE TEMPORARY ON DATABASE %I FROM PUBLIC, nexus_app, nexus_admin', current_database());
END $$;
-- +goose StatementEnd

-- H2: pre-auth функцууд зөвхөн nexus_auth
-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'nexus_auth') THEN
    RAISE EXCEPTION 'nexus_auth role байхгүй — deploy/01-roles.sql-ийн nexus_auth хэсгийг эхлээд ажиллуул';
  END IF;
END $$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION auth_user_by_email(varchar)                        FROM nexus_app, nexus_admin;
REVOKE EXECUTE ON FUNCTION auth_signup(varchar, varchar, varchar)              FROM nexus_app, nexus_admin;
REVOKE EXECUTE ON FUNCTION auth_session_create(uuid, char, timestamptz)        FROM nexus_app, nexus_admin;
REVOKE EXECUTE ON FUNCTION auth_session_lookup(char)                           FROM nexus_app, nexus_admin;
REVOKE EXECUTE ON FUNCTION auth_session_set_tenant(uuid, uuid)                 FROM nexus_app, nexus_admin;
REVOKE EXECUTE ON FUNCTION auth_session_delete(char)                           FROM nexus_app, nexus_admin;
REVOKE EXECUTE ON FUNCTION auth_sessions_purge()                               FROM nexus_app, nexus_admin;
REVOKE EXECUTE ON FUNCTION auth_handover_create(char, uuid, uuid, uuid, timestamptz) FROM nexus_app, nexus_admin;
REVOKE EXECUTE ON FUNCTION auth_handover_consume(char, char, timestamptz)      FROM nexus_app, nexus_admin;
REVOKE EXECUTE ON FUNCTION tenant_state(uuid)                                  FROM nexus_app, nexus_admin;
GRANT EXECUTE ON FUNCTION auth_user_by_email(varchar)                          TO nexus_auth;
GRANT EXECUTE ON FUNCTION auth_signup(varchar, varchar, varchar)                TO nexus_auth;
GRANT EXECUTE ON FUNCTION auth_session_create(uuid, char, timestamptz)          TO nexus_auth;
GRANT EXECUTE ON FUNCTION auth_session_lookup(char)                             TO nexus_auth;
GRANT EXECUTE ON FUNCTION auth_session_set_tenant(uuid, uuid)                   TO nexus_auth;
GRANT EXECUTE ON FUNCTION auth_session_delete(char)                             TO nexus_auth;
GRANT EXECUTE ON FUNCTION auth_sessions_purge()                                 TO nexus_auth;
GRANT EXECUTE ON FUNCTION auth_handover_create(char, uuid, uuid, uuid, timestamptz) TO nexus_auth;
GRANT EXECUTE ON FUNCTION auth_handover_consume(char, char, timestamptz)        TO nexus_auth;
GRANT EXECUTE ON FUNCTION tenant_state(uuid)                                    TO nexus_auth;
-- nexus_auth хүснэгтэд шууд хүрэхгүй — зөвхөн дээрх функцууд.
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM nexus_auth;

-- M1: ирээдүйн функцэд автомат EXECUTE өгөхгүй (REVOKE FROM PUBLIC хэвээр)
ALTER DEFAULT PRIVILEGES FOR ROLE nexus_owner IN SCHEMA public
    REVOKE EXECUTE ON FUNCTIONS FROM nexus_app, nexus_admin;

-- M3: password_hash апп role-д харагдахгүй
REVOKE SELECT ON users FROM nexus_app;
GRANT SELECT (id, email, name, platform_admin, created_at) ON users TO nexus_app;

-- M2: tenants INSERT зөвхөн id, slug, name
REVOKE INSERT ON tenants FROM nexus_app;
GRANT INSERT (id, slug, name) ON tenants TO nexus_app;

-- +goose Down
GRANT INSERT ON tenants TO nexus_app;
GRANT SELECT ON users TO nexus_app;
ALTER DEFAULT PRIVILEGES FOR ROLE nexus_owner IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO nexus_app, nexus_admin;
GRANT EXECUTE ON FUNCTION auth_user_by_email(varchar), auth_signup(varchar, varchar, varchar),
  auth_session_create(uuid, char, timestamptz), auth_session_lookup(char),
  auth_session_set_tenant(uuid, uuid), auth_session_delete(char), auth_sessions_purge(),
  auth_handover_create(char, uuid, uuid, uuid, timestamptz), auth_handover_consume(char, char, timestamptz),
  tenant_state(uuid) TO nexus_app, nexus_admin;
