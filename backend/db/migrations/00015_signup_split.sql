-- +goose Up

-- Signup нь хоёр DB role-ийн хооронд хуваагдсан (00010-ийн дараа): хэрэглэгч
-- үүсгэх нь зөвхөн nexus_auth (auth_signup), байгууллага үүсгэх нь nexus_app
-- (RLS + Go доторх role seed / permission оноолт). Нэг гүйлгээнд багтахгүй
-- тул нөхөн сэргээх (compensating) устгал: байгууллага үүсэхгүй бол дөнгөж
-- үүссэн хэрэглэгчийг буцааж устгана. Гишүүнчлэлтэй хэрэглэгчид хүрэхгүй.
-- +goose StatementBegin
CREATE FUNCTION auth_delete_tenantless_user(p_user uuid) RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, public, pg_temp AS $$
DECLARE n integer;
BEGIN
  DELETE FROM users u
   WHERE u.id = p_user
     AND NOT u.platform_admin
     AND NOT EXISTS (SELECT 1 FROM memberships m WHERE m.user_id = u.id);
  GET DIAGNOSTICS n = ROW_COUNT;
  RETURN n > 0;
END $$;
-- +goose StatementEnd
REVOKE EXECUTE ON FUNCTION auth_delete_tenantless_user(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_delete_tenantless_user(uuid) TO nexus_auth;

-- +goose Down
DROP FUNCTION auth_delete_tenantless_user(uuid);
