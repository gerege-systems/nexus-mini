-- +goose Up

-- RBAC дахин дүгнэлт (2026-08-21): membership_roles-д өөр tenant-ийн role
-- оноохыг DB давхаргад хориглоно (HTTP зам кодоор тенант дотроо хайдаг —
-- энэ нь defense-in-depth). Триггер дуудагчийн эрхээр ажиллана: nexus_app-д
-- өөр tenant-ийн role RLS-ээр харагдахгүй → NULL → IS DISTINCT FROM → алдаа.

-- +goose StatementBegin
CREATE FUNCTION membership_role_same_tenant() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog, public AS $$
BEGIN
  IF (SELECT m.tenant_id FROM memberships m WHERE m.id = NEW.membership_id)
     IS DISTINCT FROM (SELECT r.tenant_id FROM roles r WHERE r.id = NEW.role_id) THEN
    RAISE EXCEPTION 'membership_roles: role өөр tenant-ийнх' USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd

CREATE TRIGGER membership_roles_same_tenant
  BEFORE INSERT OR UPDATE ON membership_roles
  FOR EACH ROW EXECUTE FUNCTION membership_role_same_tenant();

-- +goose Down
DROP TRIGGER membership_roles_same_tenant ON membership_roles;
DROP FUNCTION membership_role_same_tenant();
