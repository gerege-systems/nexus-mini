-- +goose Up

-- Tenant админ түр нууц үгтэй данс үүсгэдэг (POST /api/members). Тэр нууц үг
-- админд мэдэгдэж байдаг тул хэрэглэгч эхний нэвтрэлтэд заавал солино —
-- солиогүй байхад tenant-ийн бүх route 403 (password_change_required).
ALTER TABLE users ADD COLUMN must_change_password boolean NOT NULL DEFAULT false;
-- Хэрэглэгч өөрийн мөрөө (RLS users_select/users_update: id = app_user_id())
-- уншиж, нууц үг солихдоо цэвэрлэнэ.
GRANT SELECT (must_change_password), UPDATE (must_change_password) ON users TO nexus_app;

-- +goose StatementBegin
CREATE FUNCTION auth_provision(p_email varchar(255), p_password_hash varchar(255), p_name varchar(120))
RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
DECLARE v_id uuid;
BEGIN
  INSERT INTO users (email, password_hash, name, must_change_password)
  VALUES (lower(p_email), p_password_hash, p_name, true)
  RETURNING users.id INTO v_id;
  RETURN v_id;
END $$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION auth_provision(varchar, varchar, varchar) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_provision(varchar, varchar, varchar) TO nexus_auth;

-- Session lookup флагийг хамт өгнө — RequireTenant хүсэлт бүрд нэмэлт
-- запросгүй шалгана.
DROP FUNCTION auth_session_lookup(char, interval);
-- +goose StatementBegin
CREATE FUNCTION auth_session_lookup(p_token_hash char(64), p_idle interval)
RETURNS TABLE (session_id uuid, user_id uuid, tenant_id uuid, platform_admin boolean,
               name varchar(120), email varchar(255), impersonated_by uuid, must_change_password boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
BEGIN
  UPDATE sessions s SET last_seen_at = clock_timestamp()
   WHERE s.token_hash = p_token_hash AND s.expires_at > clock_timestamp()
     AND s.last_seen_at > clock_timestamp() - p_idle
     AND s.last_seen_at < clock_timestamp() - interval '5 minutes';
  RETURN QUERY
  SELECT s.id, s.user_id,
         CASE WHEN s.tenant_id IS NOT NULL AND EXISTS (
                SELECT 1 FROM memberships m
                 WHERE m.tenant_id = s.tenant_id AND m.user_id = s.user_id)
              THEN s.tenant_id ELSE NULL END,
         u.platform_admin, u.name, u.email, s.impersonated_by, u.must_change_password
    FROM sessions s JOIN users u ON u.id = s.user_id
   WHERE s.token_hash = p_token_hash AND s.expires_at > clock_timestamp()
     AND s.last_seen_at > clock_timestamp() - p_idle;
END $$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION auth_session_lookup(char, interval) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_session_lookup(char, interval) TO nexus_auth;

-- +goose Down
DROP FUNCTION auth_session_lookup(char, interval);
-- +goose StatementBegin
CREATE FUNCTION auth_session_lookup(p_token_hash char(64), p_idle interval)
RETURNS TABLE (session_id uuid, user_id uuid, tenant_id uuid, platform_admin boolean,
               name varchar(120), email varchar(255), impersonated_by uuid)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
BEGIN
  UPDATE sessions s SET last_seen_at = clock_timestamp()
   WHERE s.token_hash = p_token_hash AND s.expires_at > clock_timestamp()
     AND s.last_seen_at > clock_timestamp() - p_idle
     AND s.last_seen_at < clock_timestamp() - interval '5 minutes';
  RETURN QUERY
  SELECT s.id, s.user_id,
         CASE WHEN s.tenant_id IS NOT NULL AND EXISTS (
                SELECT 1 FROM memberships m
                 WHERE m.tenant_id = s.tenant_id AND m.user_id = s.user_id)
              THEN s.tenant_id ELSE NULL END,
         u.platform_admin, u.name, u.email, s.impersonated_by
    FROM sessions s JOIN users u ON u.id = s.user_id
   WHERE s.token_hash = p_token_hash AND s.expires_at > clock_timestamp()
     AND s.last_seen_at > clock_timestamp() - p_idle;
END $$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION auth_session_lookup(char, interval) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_session_lookup(char, interval) TO nexus_auth;
DROP FUNCTION auth_provision(varchar, varchar, varchar);
ALTER TABLE users DROP COLUMN must_change_password;
