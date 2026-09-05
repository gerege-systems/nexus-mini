-- +goose Up

-- SSO (relying party): гадны issuer-ийн (iss, sub) → хэрэглэгчийн холбоос.
-- Имэйлээр байгаа данс руу холбохыг зөвхөн provider имэйлээ баталгаажуулсан
-- (email_verified=true) үед зөвшөөрнө. nexus-mini өөрөө provider болохдоо
-- email_verified=false өгдөг (signup-д имэйл баталгаажуулдаггүй) тул
-- federation-ийн хэрэглэгч энэ хүснэгтээр л танигдана — өөр issuer дээр
-- хохирогчийн имэйлээр бүртгүүлээд түүний дансанд орох боломжгүй.
CREATE TABLE sso_identities (
  issuer     varchar(255) NOT NULL,
  subject    varchar(255) NOT NULL,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (issuer, subject)
);
CREATE INDEX sso_identities_user_idx ON sso_identities (user_id);
-- 00006-ийн default эрхийг буцаана: зөвхөн доорх SECURITY DEFINER функцээр.
REVOKE ALL ON sso_identities FROM nexus_app, nexus_admin;

-- +goose StatementBegin
CREATE FUNCTION auth_sso_user(p_issuer varchar(255), p_subject varchar(255))
RETURNS TABLE (id uuid, name varchar(120), platform_admin boolean)
LANGUAGE sql SECURITY DEFINER STABLE SET search_path = pg_catalog, public, pg_temp AS $$
  SELECT u.id, u.name, u.platform_admin
    FROM sso_identities s JOIN users u ON u.id = s.user_id
   WHERE s.issuer = p_issuer AND s.subject = p_subject
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION auth_sso_link(p_issuer varchar(255), p_subject varchar(255), p_user uuid)
RETURNS void
LANGUAGE sql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
  INSERT INTO sso_identities (issuer, subject, user_id) VALUES (p_issuer, p_subject, p_user)
  ON CONFLICT (issuer, subject) DO NOTHING
$$;
-- +goose StatementEnd

-- JIT: данс + холбоос нэг гүйлгээнд — дунд нь тасарвал холбоосгүй,
-- нууц үггүй (сэргээх аргагүй) данс үлдэхгүй.
-- +goose StatementBegin
CREATE FUNCTION auth_sso_signup(p_issuer varchar(255), p_subject varchar(255),
                                p_email varchar(255), p_password_hash varchar(255), p_name varchar(120))
RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
DECLARE v_id uuid;
BEGIN
  INSERT INTO users (email, password_hash, name)
  VALUES (lower(p_email), p_password_hash, p_name)
  RETURNING users.id INTO v_id;
  INSERT INTO sso_identities (issuer, subject, user_id) VALUES (p_issuer, p_subject, v_id);
  RETURN v_id;
END $$;
-- +goose StatementEnd

REVOKE EXECUTE ON FUNCTION auth_sso_user(varchar, varchar), auth_sso_link(varchar, varchar, uuid),
  auth_sso_signup(varchar, varchar, varchar, varchar, varchar) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_sso_user(varchar, varchar), auth_sso_link(varchar, varchar, uuid),
  auth_sso_signup(varchar, varchar, varchar, varchar, varchar) TO nexus_auth;

-- +goose Down
DROP FUNCTION auth_sso_signup(varchar, varchar, varchar, varchar, varchar);
DROP FUNCTION auth_sso_link(varchar, varchar, uuid);
DROP FUNCTION auth_sso_user(varchar, varchar);
DROP TABLE sso_identities;
