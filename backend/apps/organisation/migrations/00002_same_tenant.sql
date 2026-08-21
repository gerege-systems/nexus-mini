-- +goose Up

-- FK шалгалт RLS-ийг давдаг тул өөр tenant-ийн membership/department-ийг
-- uuid таамаглаж холбож болдог байв — DB давхаргад хориглоно (цөмийн 00007-той
-- ижил загвар: дуудагчийн эрхээр, харагдахгүй мөр → NULL → IS DISTINCT FROM).
-- +goose StatementBegin
CREATE FUNCTION org_same_tenant() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog, public, pg_temp AS $$
BEGIN
  IF TG_TABLE_NAME = 'org_departments' THEN
    IF NEW.manager_membership_id IS NOT NULL AND
       (SELECT m.tenant_id FROM memberships m WHERE m.id = NEW.manager_membership_id) IS DISTINCT FROM NEW.tenant_id THEN
      RAISE EXCEPTION 'org_departments: менежер өөр tenant-ийнх' USING ERRCODE = '23514';
    END IF;
    IF NEW.parent_id IS NOT NULL AND
       (SELECT d.tenant_id FROM org_departments d WHERE d.id = NEW.parent_id) IS DISTINCT FROM NEW.tenant_id THEN
      RAISE EXCEPTION 'org_departments: дээд нэгж өөр tenant-ийнх' USING ERRCODE = '23514';
    END IF;
  ELSE
    IF (SELECT m.tenant_id FROM memberships m WHERE m.id = NEW.membership_id) IS DISTINCT FROM NEW.tenant_id THEN
      RAISE EXCEPTION 'org_positions: гишүүнчлэл өөр tenant-ийнх' USING ERRCODE = '23514';
    END IF;
    IF NEW.department_id IS NOT NULL AND
       (SELECT d.tenant_id FROM org_departments d WHERE d.id = NEW.department_id) IS DISTINCT FROM NEW.tenant_id THEN
      RAISE EXCEPTION 'org_positions: хэлтэс өөр tenant-ийнх' USING ERRCODE = '23514';
    END IF;
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER org_departments_same_tenant BEFORE INSERT OR UPDATE ON org_departments
  FOR EACH ROW EXECUTE FUNCTION org_same_tenant();
CREATE TRIGGER org_positions_same_tenant BEFORE INSERT OR UPDATE ON org_positions
  FOR EACH ROW EXECUTE FUNCTION org_same_tenant();
CREATE INDEX idx_org_departments_manager ON org_departments(manager_membership_id);
CREATE INDEX idx_org_positions_department ON org_positions(department_id);

-- +goose Down
DROP TRIGGER org_positions_same_tenant ON org_positions;
DROP TRIGGER org_departments_same_tenant ON org_departments;
DROP FUNCTION org_same_tenant();
DROP INDEX idx_org_departments_manager;
DROP INDEX idx_org_positions_department;
