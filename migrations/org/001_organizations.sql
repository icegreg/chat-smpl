-- Organization Service Schema
-- Schema: con_test

-- Companies (головная и дочерние компании)
CREATE TABLE IF NOT EXISTS con_test.companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id UUID REFERENCES con_test.companies(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(100),
    description TEXT,
    instance_id UUID,  -- Подготовка для федерации
    timezone VARCHAR(50) DEFAULT 'Europe/Moscow',  -- UTC+3 по умолчанию
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_companies_parent_id ON con_test.companies(parent_id);
CREATE INDEX IF NOT EXISTS idx_companies_instance_id ON con_test.companies(instance_id) WHERE instance_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_companies_is_active ON con_test.companies(is_active);

-- Departments (структурные подразделения)
CREATE TABLE IF NOT EXISTS con_test.departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES con_test.companies(id) ON DELETE CASCADE,
    parent_department_id UUID REFERENCES con_test.departments(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(100),
    description TEXT,
    instance_id UUID,  -- Подготовка для федерации
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_departments_company_id ON con_test.departments(company_id);
CREATE INDEX IF NOT EXISTS idx_departments_parent_id ON con_test.departments(parent_department_id);
CREATE INDEX IF NOT EXISTS idx_departments_instance_id ON con_test.departments(instance_id) WHERE instance_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_departments_is_active ON con_test.departments(is_active);

-- Positions (должности)
CREATE TABLE IF NOT EXISTS con_test.positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES con_test.companies(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(100),
    level INT DEFAULT 0,  -- Уровень должности (0-самый низкий, 100-самый высокий)
    description TEXT,
    instance_id UUID,  -- Подготовка для федерации
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_positions_company_id ON con_test.positions(company_id);
CREATE INDEX IF NOT EXISTS idx_positions_level ON con_test.positions(level);
CREATE INDEX IF NOT EXISTS idx_positions_instance_id ON con_test.positions(instance_id) WHERE instance_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_positions_is_active ON con_test.positions(is_active);

-- Employees (привязка пользователей к организации)
CREATE TABLE IF NOT EXISTS con_test.employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES con_test.users(id) ON DELETE CASCADE,
    department_id UUID NOT NULL REFERENCES con_test.departments(id) ON DELETE CASCADE,
    position_id UUID NOT NULL REFERENCES con_test.positions(id) ON DELETE RESTRICT,
    employee_number VARCHAR(50),  -- Табельный номер
    hire_date DATE,
    instance_id UUID,  -- Подготовка для федерации
    is_primary BOOLEAN DEFAULT true,  -- Основное место работы
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, department_id)  -- Пользователь может быть в подразделении только один раз
);

CREATE INDEX IF NOT EXISTS idx_employees_user_id ON con_test.employees(user_id);
CREATE INDEX IF NOT EXISTS idx_employees_department_id ON con_test.employees(department_id);
CREATE INDEX IF NOT EXISTS idx_employees_position_id ON con_test.employees(position_id);
CREATE INDEX IF NOT EXISTS idx_employees_instance_id ON con_test.employees(instance_id) WHERE instance_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_employees_is_active ON con_test.employees(is_active);
CREATE INDEX IF NOT EXISTS idx_employees_is_primary ON con_test.employees(is_primary) WHERE is_primary = true;

-- Триггеры для updated_at
CREATE OR REPLACE FUNCTION con_test.update_org_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_companies_updated_at ON con_test.companies;
CREATE TRIGGER trg_companies_updated_at
    BEFORE UPDATE ON con_test.companies
    FOR EACH ROW EXECUTE FUNCTION con_test.update_org_updated_at();

DROP TRIGGER IF EXISTS trg_departments_updated_at ON con_test.departments;
CREATE TRIGGER trg_departments_updated_at
    BEFORE UPDATE ON con_test.departments
    FOR EACH ROW EXECUTE FUNCTION con_test.update_org_updated_at();

DROP TRIGGER IF EXISTS trg_positions_updated_at ON con_test.positions;
CREATE TRIGGER trg_positions_updated_at
    BEFORE UPDATE ON con_test.positions
    FOR EACH ROW EXECUTE FUNCTION con_test.update_org_updated_at();

DROP TRIGGER IF EXISTS trg_employees_updated_at ON con_test.employees;
CREATE TRIGGER trg_employees_updated_at
    BEFORE UPDATE ON con_test.employees
    FOR EACH ROW EXECUTE FUNCTION con_test.update_org_updated_at();
