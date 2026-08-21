-- +goose Up

-- Аудит 2026-08-22: handover хэрэглэх мөчид дахин шалгана (админ мөн
-- platform_admin хэвээр юу, бай platform_admin болчихоогүй юу, гишүүнчлэл
-- хэвээр юу) ба tenant/user/admin-ийг буцаана (Go талд audit бичихэд).
DROP FUNCTION auth_handover_consume(char, char, timestamptz);
-- +goose StatementBegin
CREATE FUNCTION auth_handover_consume(
    p_token_hash char(64), p_session_hash char(64), p_session_expires timestamptz)
RETURNS TABLE (session_id uuid, tenant_id uuid, user_id uuid, admin_id uuid)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
DECLARE v_user uuid; v_tenant uuid; v_admin uuid; v_sid uuid;
BEGIN
  DELETE FROM handover_tokens h
   WHERE h.token_hash = p_token_hash AND h.expires_at > clock_timestamp()
  RETURNING h.user_id, h.tenant_id, h.admin_id INTO v_user, v_tenant, v_admin;
  IF v_user IS NULL THEN
    RETURN;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM users u WHERE u.id = v_admin AND u.platform_admin)
     OR EXISTS (SELECT 1 FROM users u WHERE u.id = v_user AND u.platform_admin)
     OR NOT EXISTS (SELECT 1 FROM memberships m WHERE m.tenant_id = v_tenant AND m.user_id = v_user) THEN
    RETURN;
  END IF;
  INSERT INTO sessions (user_id, tenant_id, token_hash, expires_at, impersonated_by)
  VALUES (v_user, v_tenant, p_session_hash, p_session_expires, v_admin)
  RETURNING id INTO v_sid;
  RETURN QUERY SELECT v_sid, v_tenant, v_user, v_admin;
END $$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION auth_handover_consume(char, char, timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_handover_consume(char, char, timestamptz) TO nexus_auth;

-- +goose Down
DROP FUNCTION auth_handover_consume(char, char, timestamptz);
-- +goose StatementBegin
CREATE FUNCTION auth_handover_consume(
    p_token_hash char(64), p_session_hash char(64), p_session_expires timestamptz)
RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
DECLARE v_user uuid; v_tenant uuid; v_admin uuid; v_sid uuid;
BEGIN
  DELETE FROM handover_tokens
   WHERE token_hash = p_token_hash AND expires_at > clock_timestamp()
  RETURNING user_id, tenant_id, admin_id INTO v_user, v_tenant, v_admin;
  IF v_user IS NULL THEN RETURN NULL; END IF;
  INSERT INTO sessions (user_id, tenant_id, token_hash, expires_at, impersonated_by)
  VALUES (v_user, v_tenant, p_session_hash, p_session_expires, v_admin) RETURNING id INTO v_sid;
  RETURN v_sid;
END $$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION auth_handover_consume(char, char, timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_handover_consume(char, char, timestamptz) TO nexus_auth;
