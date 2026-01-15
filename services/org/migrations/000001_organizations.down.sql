-- Rollback org migrations
DROP TRIGGER IF EXISTS trg_employees_updated_at ON con_test.employees;
DROP TRIGGER IF EXISTS trg_positions_updated_at ON con_test.positions;
DROP TRIGGER IF EXISTS trg_departments_updated_at ON con_test.departments;
DROP TRIGGER IF EXISTS trg_companies_updated_at ON con_test.companies;
DROP FUNCTION IF EXISTS con_test.update_org_updated_at();
DROP TABLE IF EXISTS con_test.employees;
DROP TABLE IF EXISTS con_test.positions;
DROP TABLE IF EXISTS con_test.departments;
DROP TABLE IF EXISTS con_test.companies;
