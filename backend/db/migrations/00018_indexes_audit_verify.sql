-- +goose Up

-- FK / шүүлтийн индексүүд: cascade устгал, tenant-ийн session revoke,
-- oauth_client_cleanup, гишүүн хасахад seq scan хийдэг байв.
CREATE INDEX idx_sessions_tenant           ON sessions (tenant_id);
CREATE INDEX idx_sessions_impersonated_by  ON sessions (impersonated_by);
CREATE INDEX idx_oauth_tokens_user         ON oauth_tokens (user_id);
CREATE INDEX idx_oauth_tokens_client       ON oauth_tokens (client_id);
CREATE INDEX idx_oauth_codes_user          ON oauth_codes (user_id);
CREATE INDEX idx_membership_roles_role     ON membership_roles (role_id);
CREATE INDEX idx_app_installations_app     ON app_installations (app_id);
CREATE INDEX idx_installation_events_user  ON installation_events (user_id);
CREATE INDEX idx_oauth_clients_created_by  ON oauth_clients (created_by);

-- audit_verify: audit_append-тэй адил зөвхөн өөрийн tenant (эсвэл платформ).
-- Өмнө nexus_app дурын tenant id-гаар дуудаж өөр tenant-ийн гинж эвдэрсэн
-- эсэх / мөрийн id-г мэдэж чадаж байв.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION audit_verify(p_tenant uuid)
RETURNS bigint
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
DECLARE
  r      record;
  v_prev char(64) := repeat('0', 64);
BEGIN
  IF p_tenant IS DISTINCT FROM app_tenant_id()
     AND NOT pg_has_role(session_user, 'nexus_platform', 'member') THEN
    RAISE EXCEPTION 'audit: tenant context зөрүүтэй';
  END IF;
  FOR r IN SELECT * FROM audit_log WHERE tenant_id = p_tenant ORDER BY id LOOP
    IF r.prev_hash <> v_prev
       OR r.hash <> audit_row_hash(r.prev_hash, r.tenant_id, r.user_id,
                                   r.action, r.object, r.details, r.occurred_at) THEN
      RETURN r.id;
    END IF;
    v_prev := r.hash;
  END LOOP;
  RETURN NULL;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION audit_verify(p_tenant uuid)
RETURNS bigint
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog, public AS $$
DECLARE
  r      record;
  v_prev char(64) := repeat('0', 64);
BEGIN
  FOR r IN SELECT * FROM audit_log WHERE tenant_id = p_tenant ORDER BY id LOOP
    IF r.prev_hash <> v_prev
       OR r.hash <> audit_row_hash(r.prev_hash, r.tenant_id, r.user_id,
                                   r.action, r.object, r.details, r.occurred_at) THEN
      RETURN r.id;
    END IF;
    v_prev := r.hash;
  END LOOP;
  RETURN NULL;
END $$;
-- +goose StatementEnd
DROP INDEX idx_oauth_clients_created_by;
DROP INDEX idx_installation_events_user;
DROP INDEX idx_app_installations_app;
DROP INDEX idx_membership_roles_role;
DROP INDEX idx_oauth_codes_user;
DROP INDEX idx_oauth_tokens_client;
DROP INDEX idx_oauth_tokens_user;
DROP INDEX idx_sessions_impersonated_by;
DROP INDEX idx_sessions_tenant;
