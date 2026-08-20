-- +goose Up

-- Audit — append-only, tenant тус бүрийн hash chain.
-- Сургамж (docs/01-lessons.md #2): hash-ийг DB дотор тооцно (Go талд
-- тооцвол jsonb хэвшүүлэлтээс болж хэзээ ч тохирохгүй), цагийг advisory
-- lock ДОТОР clock_timestamp()-ээр авна (бичигдэх дараалалтай зөрөхгүй).

CREATE TABLE audit_log (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id   uuid NOT NULL,
    user_id     uuid,
    action      varchar(128) NOT NULL,
    object      varchar(255) NOT NULL DEFAULT '',
    details     jsonb NOT NULL DEFAULT '{}',
    prev_hash   char(64) NOT NULL,
    hash        char(64) NOT NULL,
    occurred_at timestamptz NOT NULL
);
CREATE INDEX idx_audit_tenant ON audit_log(tenant_id, id DESC);

ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY audit_select ON audit_log FOR SELECT USING (
    tenant_id = app_tenant_id() OR app_is_platform()
);
-- INSERT бодлого зориуд байхгүй — бичих цорын ганц зам нь audit_append().

-- Update/delete-ийг owner-оос ч хамгаална.
-- +goose StatementBegin
CREATE FUNCTION audit_block_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'audit_log is append-only';
END $$;
-- +goose StatementEnd

CREATE TRIGGER audit_no_update BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_block_mutation();

-- +goose StatementBegin
CREATE FUNCTION audit_row_hash(
    p_prev char(64), p_tenant uuid, p_user uuid, p_action varchar(128),
    p_object varchar(255), p_details jsonb, p_at timestamptz)
RETURNS char(64)
LANGUAGE sql IMMUTABLE AS $$
  SELECT encode(sha256(convert_to(
    p_prev || ':' || p_tenant::text || ':' || coalesce(p_user::text, '') || ':' ||
    p_action || ':' || p_object || ':' || p_details::text || ':' ||
    to_char(p_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
    'UTF8')), 'hex')
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION audit_append(
    p_tenant uuid, p_user uuid, p_action varchar(128),
    p_object varchar(255), p_details jsonb)
RETURNS bigint
LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
  v_prev char(64);
  v_at   timestamptz;
  v_id   bigint;
BEGIN
  -- Tenant тус бүрийн гинжийг advisory lock-оор цувуулна.
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

-- Гинжийг шалгах: эвдэрсэн эхний мөрийн id, бүрэн бол NULL.
-- +goose StatementBegin
CREATE FUNCTION audit_verify(p_tenant uuid)
RETURNS bigint
LANGUAGE plpgsql STABLE SECURITY DEFINER AS $$
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

GRANT SELECT ON audit_log TO nexus_app, nexus_admin;
GRANT EXECUTE ON FUNCTION audit_append(uuid, uuid, varchar, varchar, jsonb) TO nexus_app, nexus_admin;
GRANT EXECUTE ON FUNCTION audit_verify(uuid) TO nexus_app, nexus_admin;

-- +goose Down
DROP FUNCTION audit_verify(uuid);
DROP FUNCTION audit_append(uuid, uuid, varchar, varchar, jsonb);
DROP FUNCTION audit_row_hash(char, uuid, uuid, varchar, varchar, jsonb, timestamptz);
DROP TRIGGER audit_no_update ON audit_log;
DROP FUNCTION audit_block_mutation();
DROP TABLE audit_log;
