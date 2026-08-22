-- +goose Up

-- OGN цөмтэй харьцуулсны дараах 5 жижиг цоорхой (2026-08-22):
--  1. Дансны түр түгжээ: 5 буруу оролдлого (15 мин дотор) → 15 мин түгжээ.
--     IP rate limit-ээс тусдаа — тархсан brute force-оос хамгаална.
--  2. Session idle timeout: сүүлийн хэрэглээнээс 90 мин хэрэглээгүй бол дуусна.
--  5. Түдгэлзүүлэхэд tenant-ийн бүх session шууд хүчингүй.
ALTER TABLE users
    ADD COLUMN failed_login_attempts smallint NOT NULL DEFAULT 0,
    ADD COLUMN last_failed_at timestamptz,
    ADD COLUMN locked_until timestamptz;
ALTER TABLE sessions ADD COLUMN last_seen_at timestamptz NOT NULL DEFAULT now();

-- +goose StatementBegin
-- Түгжээтэй бол locked_until буцаана, үгүй бол NULL.
CREATE FUNCTION auth_lockout(p_email varchar(255)) RETURNS timestamptz
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public, pg_temp AS $$
  SELECT u.locked_until FROM users u
   WHERE u.email = lower(p_email) AND u.locked_until > clock_timestamp()
$$;
-- +goose StatementEnd
-- +goose StatementBegin
-- Нэвтрэлтийн үр дүнг бүртгэнэ: амжилт → тоолуур 0; алдаа → +1, 15 мин
-- цонх дууссан бол шинээр эхэлнэ; 5 хүрвэл 15 мин түгжинэ. Буцаах: түгжээ
-- эхэлсэн эсэх.
CREATE FUNCTION auth_login_result(p_email varchar(255), p_ok boolean) RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
DECLARE v_n smallint; v_locked boolean := false;
BEGIN
  IF p_ok THEN
    UPDATE users SET failed_login_attempts = 0, last_failed_at = NULL, locked_until = NULL
     WHERE email = lower(p_email);
    RETURN false;
  END IF;
  UPDATE users
     SET failed_login_attempts = CASE WHEN last_failed_at IS NULL OR last_failed_at < clock_timestamp() - interval '15 minutes'
                                      THEN 1 ELSE failed_login_attempts + 1 END,
         last_failed_at = clock_timestamp()
   WHERE email = lower(p_email)
  RETURNING failed_login_attempts INTO v_n;
  IF v_n >= 5 THEN
    UPDATE users SET locked_until = clock_timestamp() + interval '15 minutes', failed_login_attempts = 0
     WHERE email = lower(p_email);
    v_locked := true;
  END IF;
  RETURN v_locked;
END $$;
-- +goose StatementEnd

-- 2. lookup: idle timeout + last_seen (5 минутад нэгээс олон бичихгүй).
DROP FUNCTION auth_session_lookup(char);
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

-- 5. Tenant-ийн бүх session-ийг хүчингүй болгоно (түдгэлзүүлэхэд).
-- +goose StatementBegin
CREATE FUNCTION auth_sessions_revoke_tenant(p_tenant uuid) RETURNS integer
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
DECLARE n integer;
BEGIN
  DELETE FROM sessions WHERE tenant_id = p_tenant;
  GET DIAGNOSTICS n = ROW_COUNT;
  RETURN n;
END $$;
-- +goose StatementEnd

REVOKE EXECUTE ON FUNCTION auth_lockout(varchar), auth_login_result(varchar, boolean),
  auth_session_lookup(char, interval), auth_sessions_revoke_tenant(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_lockout(varchar), auth_login_result(varchar, boolean),
  auth_session_lookup(char, interval), auth_sessions_revoke_tenant(uuid) TO nexus_auth;

-- +goose Down
DROP FUNCTION auth_sessions_revoke_tenant(uuid);
DROP FUNCTION auth_session_lookup(char, interval);
-- +goose StatementBegin
CREATE FUNCTION auth_session_lookup(p_token_hash char(64))
RETURNS TABLE (session_id uuid, user_id uuid, tenant_id uuid, platform_admin boolean,
               name varchar(120), email varchar(255), impersonated_by uuid)
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public, pg_temp AS $$
  SELECT s.id, s.user_id,
         CASE WHEN s.tenant_id IS NOT NULL AND EXISTS (
                SELECT 1 FROM memberships m WHERE m.tenant_id = s.tenant_id AND m.user_id = s.user_id)
              THEN s.tenant_id ELSE NULL END,
         u.platform_admin, u.name, u.email, s.impersonated_by
    FROM sessions s JOIN users u ON u.id = s.user_id
   WHERE s.token_hash = p_token_hash AND s.expires_at > now()
$$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION auth_session_lookup(char) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_session_lookup(char) TO nexus_auth;
DROP FUNCTION auth_login_result(varchar, boolean);
DROP FUNCTION auth_lockout(varchar);
ALTER TABLE sessions DROP COLUMN last_seen_at;
ALTER TABLE users DROP COLUMN failed_login_attempts, DROP COLUMN last_failed_at, DROP COLUMN locked_until;
