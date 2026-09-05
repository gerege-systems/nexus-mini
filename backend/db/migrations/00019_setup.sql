-- +goose Up

-- Анхны тохиргооны шидтэн (/setup, open-gerege-nexus-ийн загвар): вэбээс
-- платформын анхны админыг үүсгэнэ. Хамгаалалт DB талд: платформ админ аль
-- хэдийн байвал NULL — handler-ийн алдаа ч хоёр дахь админ үүсгэж чадахгүй.
-- Advisory lock: зэрэг ирсэн хоёр хүсэлтийн зөвхөн нэг нь амжилттай.
-- +goose StatementBegin
CREATE FUNCTION auth_setup_admin(p_email varchar(255), p_password_hash varchar(255), p_name varchar(120))
RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
DECLARE v_id uuid;
BEGIN
  PERFORM pg_advisory_xact_lock(hashtext('nexus-mini:setup'));
  IF EXISTS (SELECT 1 FROM users u WHERE u.platform_admin) THEN
    RETURN NULL;
  END IF;
  INSERT INTO users (email, password_hash, name, platform_admin)
  VALUES (lower(p_email), p_password_hash, p_name, true)
  RETURNING users.id INTO v_id;
  RETURN v_id;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION auth_platform_admin_exists() RETURNS boolean
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM users u WHERE u.platform_admin)
$$;
-- +goose StatementEnd

REVOKE EXECUTE ON FUNCTION auth_setup_admin(varchar, varchar, varchar), auth_platform_admin_exists() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_setup_admin(varchar, varchar, varchar), auth_platform_admin_exists() TO nexus_auth;

-- +goose Down
DROP FUNCTION auth_platform_admin_exists();
DROP FUNCTION auth_setup_admin(varchar, varchar, varchar);
