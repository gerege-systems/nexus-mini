-- +goose Up

-- Платформын админы impersonation (2026-08-21): админ панель (тусдаа домэйн)
-- нэг удаагийн 60с handover token үүсгэж, portal дээр 30 мин-ын
-- impersonated_by тэмдэгтэй session болгоно. Бүх шалгалт DB талд.

ALTER TABLE sessions ADD COLUMN impersonated_by uuid REFERENCES users(id) ON DELETE CASCADE;

CREATE TABLE handover_tokens (
    token_hash char(64) PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    admin_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
-- Policy-гүй RLS = апп/админ role шууд уншиж/бичихгүй; зөвхөн доорх
-- SECURITY DEFINER функцууд (owner RLS-ээс чөлөөтэй).
ALTER TABLE handover_tokens ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON handover_tokens FROM nexus_app, nexus_admin;

-- +goose StatementBegin
CREATE FUNCTION auth_handover_create(
    p_token_hash char(64), p_user uuid, p_tenant uuid, p_admin uuid, p_expires timestamptz)
RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public AS $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM users WHERE id = p_admin AND platform_admin) THEN
    RAISE EXCEPTION 'handover: платформын админ биш' USING ERRCODE = '42501';
  END IF;
  IF EXISTS (SELECT 1 FROM users WHERE id = p_user AND platform_admin) THEN
    RAISE EXCEPTION 'handover: платформын админы нэрийн өмнөөс орохгүй' USING ERRCODE = '42501';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM memberships WHERE tenant_id = p_tenant AND user_id = p_user) THEN
    RAISE EXCEPTION 'handover: гишүүнчлэл байхгүй' USING ERRCODE = '42501';
  END IF;
  -- Хуучирсан token-уудыг зэрэг цэвэрлэнэ (жижиг хүснэгт).
  DELETE FROM handover_tokens WHERE expires_at < clock_timestamp();
  INSERT INTO handover_tokens (token_hash, user_id, tenant_id, admin_id, expires_at)
  VALUES (p_token_hash, p_user, p_tenant, p_admin, p_expires);
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION auth_handover_consume(
    p_token_hash char(64), p_session_hash char(64), p_session_expires timestamptz)
RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public AS $$
DECLARE v_user uuid; v_tenant uuid; v_admin uuid; v_sid uuid;
BEGIN
  DELETE FROM handover_tokens
   WHERE token_hash = p_token_hash AND expires_at > clock_timestamp()
  RETURNING user_id, tenant_id, admin_id INTO v_user, v_tenant, v_admin;
  IF v_user IS NULL THEN
    RETURN NULL;
  END IF;
  INSERT INTO sessions (user_id, tenant_id, token_hash, expires_at, impersonated_by)
  VALUES (v_user, v_tenant, p_session_hash, p_session_expires, v_admin)
  RETURNING id INTO v_sid;
  RETURN v_sid;
END $$;
-- +goose StatementEnd

-- lookup: impersonated_by нэмэгдсэн (буцаах төрөл өөрчлөгдөх тул DROP+CREATE).
DROP FUNCTION auth_session_lookup(char);
-- +goose StatementBegin
CREATE FUNCTION auth_session_lookup(p_token_hash char(64))
RETURNS TABLE (session_id uuid, user_id uuid, tenant_id uuid, platform_admin boolean,
               name varchar(120), email varchar(255), impersonated_by uuid)
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public AS $$
  SELECT s.id, s.user_id,
         CASE WHEN s.tenant_id IS NOT NULL AND EXISTS (
                SELECT 1 FROM memberships m
                 WHERE m.tenant_id = s.tenant_id AND m.user_id = s.user_id)
              THEN s.tenant_id ELSE NULL END,
         u.platform_admin, u.name, u.email, s.impersonated_by
    FROM sessions s JOIN users u ON u.id = s.user_id
   WHERE s.token_hash = p_token_hash AND s.expires_at > now()
$$;
-- +goose StatementEnd

REVOKE EXECUTE ON FUNCTION auth_handover_create(char, uuid, uuid, uuid, timestamptz) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION auth_handover_consume(char, char, timestamptz) FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION auth_session_lookup(char) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_handover_create(char, uuid, uuid, uuid, timestamptz) TO nexus_app, nexus_admin;
GRANT EXECUTE ON FUNCTION auth_handover_consume(char, char, timestamptz) TO nexus_app, nexus_admin;
GRANT EXECUTE ON FUNCTION auth_session_lookup(char) TO nexus_app, nexus_admin;

-- +goose Down
DROP FUNCTION auth_session_lookup(char);
-- +goose StatementBegin
CREATE FUNCTION auth_session_lookup(p_token_hash char(64))
RETURNS TABLE (session_id uuid, user_id uuid, tenant_id uuid, platform_admin boolean, name varchar(120), email varchar(255))
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public AS $$
  SELECT s.id, s.user_id,
         CASE WHEN s.tenant_id IS NOT NULL AND EXISTS (
                SELECT 1 FROM memberships m
                 WHERE m.tenant_id = s.tenant_id AND m.user_id = s.user_id)
              THEN s.tenant_id ELSE NULL END,
         u.platform_admin, u.name, u.email
    FROM sessions s JOIN users u ON u.id = s.user_id
   WHERE s.token_hash = p_token_hash AND s.expires_at > now()
$$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION auth_session_lookup(char) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_session_lookup(char) TO nexus_app, nexus_admin;
DROP FUNCTION auth_handover_consume(char, char, timestamptz);
DROP FUNCTION auth_handover_create(char, uuid, uuid, uuid, timestamptz);
DROP TABLE handover_tokens;
ALTER TABLE sessions DROP COLUMN impersonated_by;
