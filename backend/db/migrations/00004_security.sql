-- +goose Up

-- Аюулгүй байдлын аудитын засварууд (2026-08-20).

-- #2: Хасагдсан гишүүний session tenant-даа үлддэг байсан — lookup нь
-- гишүүнчлэлийг шалгаж, байхгүй бол tenant NULL буцаана (дараагийн
-- хүсэлтээс эхлэн тухайн байгууллагын юу ч харагдахгүй).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION auth_session_lookup(p_token_hash char(64))
RETURNS TABLE (session_id uuid, user_id uuid, tenant_id uuid, platform_admin boolean, name varchar(120), email varchar(255))
LANGUAGE sql SECURITY DEFINER STABLE AS $$
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

-- #3: users UPDATE — аппын role зөвхөн өөрийн нэр/нууц үгээ солино.
-- Багана хязгаарлаагүй бол ирээдүйн profile handler platform_admin=true
-- тавих цоорхой болно.
DROP POLICY users_update ON users;
CREATE POLICY users_update ON users FOR UPDATE
    USING (id = app_user_id() OR app_is_platform())
    WITH CHECK (id = app_user_id() OR app_is_platform());
REVOKE UPDATE ON users FROM nexus_app;
GRANT UPDATE (name, password_hash) ON users TO nexus_app;

-- +goose Down
REVOKE UPDATE (name, password_hash) ON users FROM nexus_app;
GRANT UPDATE ON users TO nexus_app;
DROP POLICY users_update ON users;
CREATE POLICY users_update ON users FOR UPDATE
    USING (id = app_user_id() OR app_is_platform());
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION auth_session_lookup(p_token_hash char(64))
RETURNS TABLE (session_id uuid, user_id uuid, tenant_id uuid, platform_admin boolean, name varchar(120), email varchar(255))
LANGUAGE sql SECURITY DEFINER STABLE AS $$
  SELECT s.id, s.user_id, s.tenant_id, u.platform_admin, u.name, u.email
    FROM sessions s JOIN users u ON u.id = s.user_id
   WHERE s.token_hash = p_token_hash AND s.expires_at > now()
$$;
-- +goose StatementEnd
