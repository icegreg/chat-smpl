package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/icegreg/chat-smpl/services/org/internal/model"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

type OrgRepository interface {
	// Companies
	CreateCompany(ctx context.Context, c *model.Company) error
	GetCompany(ctx context.Context, id uuid.UUID) (*model.Company, error)
	ListCompanies(ctx context.Context, parentID *uuid.UUID, includeInactive bool, page, count int) ([]model.Company, int, error)
	UpdateCompany(ctx context.Context, c *model.Company) error
	DeleteCompany(ctx context.Context, id uuid.UUID) error

	// Departments
	CreateDepartment(ctx context.Context, d *model.Department) error
	GetDepartment(ctx context.Context, id uuid.UUID) (*model.Department, error)
	ListDepartments(ctx context.Context, companyID uuid.UUID, parentID *uuid.UUID, includeInactive bool, page, count int) ([]model.Department, int, error)
	UpdateDepartment(ctx context.Context, d *model.Department) error
	DeleteDepartment(ctx context.Context, id uuid.UUID) error

	// Positions
	CreatePosition(ctx context.Context, p *model.Position) error
	GetPosition(ctx context.Context, id uuid.UUID) (*model.Position, error)
	ListPositions(ctx context.Context, companyID uuid.UUID, includeInactive bool, page, count int) ([]model.Position, int, error)
	UpdatePosition(ctx context.Context, p *model.Position) error
	DeletePosition(ctx context.Context, id uuid.UUID) error

	// Employees
	CreateEmployee(ctx context.Context, e *model.Employee) error
	GetEmployee(ctx context.Context, id uuid.UUID) (*model.Employee, error)
	GetEmployeeByUserID(ctx context.Context, userID uuid.UUID, primaryOnly bool) (*model.Employee, error)
	ListEmployees(ctx context.Context, departmentID, companyID *uuid.UUID, includeInactive bool, page, count int) ([]model.Employee, int, error)
	UpdateEmployee(ctx context.Context, e *model.Employee) error
	DeleteEmployee(ctx context.Context, id uuid.UUID) error

	// Enrichment
	GetUserOrgInfo(ctx context.Context, userID uuid.UUID) (*model.UserOrgInfo, error)
	GetUsersOrgInfoBatch(ctx context.Context, userIDs []uuid.UUID) ([]model.UserOrgInfo, error)
}

type orgRepository struct {
	pool *pgxpool.Pool
}

func NewOrgRepository(pool *pgxpool.Pool) OrgRepository {
	return &orgRepository{pool: pool}
}

// ==================== Companies ====================

func (r *orgRepository) CreateCompany(ctx context.Context, c *model.Company) error {
	query := `
		INSERT INTO con_test.companies (id, parent_id, name, short_name, description, timezone, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.Timezone == "" {
		c.Timezone = "Europe/Moscow"
	}
	err := r.pool.QueryRow(ctx, query, c.ID, c.ParentID, c.Name, c.ShortName, c.Description, c.Timezone, c.IsActive).
		Scan(&c.CreatedAt, &c.UpdatedAt)
	return err
}

func (r *orgRepository) GetCompany(ctx context.Context, id uuid.UUID) (*model.Company, error) {
	query := `
		SELECT id, parent_id, name, short_name, description, instance_id, timezone, is_active, created_at, updated_at
		FROM con_test.companies WHERE id = $1
	`
	var c model.Company
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.ParentID, &c.Name, &c.ShortName, &c.Description,
		&c.InstanceID, &c.Timezone, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (r *orgRepository) ListCompanies(ctx context.Context, parentID *uuid.UUID, includeInactive bool, page, count int) ([]model.Company, int, error) {
	offset := (page - 1) * count
	if offset < 0 {
		offset = 0
	}

	var whereClause string
	var args []interface{}
	argNum := 1

	if parentID != nil {
		whereClause = fmt.Sprintf("WHERE parent_id = $%d", argNum)
		args = append(args, *parentID)
		argNum++
	} else {
		whereClause = "WHERE parent_id IS NULL"
	}

	if !includeInactive {
		whereClause += " AND is_active = true"
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM con_test.companies %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// List
	query := fmt.Sprintf(`
		SELECT id, parent_id, name, short_name, description, instance_id, timezone, is_active, created_at, updated_at
		FROM con_test.companies %s
		ORDER BY name
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, count, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var companies []model.Company
	for rows.Next() {
		var c model.Company
		if err := rows.Scan(&c.ID, &c.ParentID, &c.Name, &c.ShortName, &c.Description,
			&c.InstanceID, &c.Timezone, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		companies = append(companies, c)
	}
	return companies, total, nil
}

func (r *orgRepository) UpdateCompany(ctx context.Context, c *model.Company) error {
	query := `
		UPDATE con_test.companies
		SET name = $2, short_name = $3, description = $4, timezone = $5, is_active = $6
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, c.ID, c.Name, c.ShortName, c.Description, c.Timezone, c.IsActive).
		Scan(&c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *orgRepository) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM con_test.companies WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== Departments ====================

func (r *orgRepository) CreateDepartment(ctx context.Context, d *model.Department) error {
	query := `
		INSERT INTO con_test.departments (id, company_id, parent_department_id, name, short_name, description, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	err := r.pool.QueryRow(ctx, query, d.ID, d.CompanyID, d.ParentDepartmentID, d.Name, d.ShortName, d.Description, d.SortOrder, d.IsActive).
		Scan(&d.CreatedAt, &d.UpdatedAt)
	return err
}

func (r *orgRepository) GetDepartment(ctx context.Context, id uuid.UUID) (*model.Department, error) {
	query := `
		SELECT id, company_id, parent_department_id, name, short_name, description, instance_id, sort_order, is_active, created_at, updated_at
		FROM con_test.departments WHERE id = $1
	`
	var d model.Department
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.CompanyID, &d.ParentDepartmentID, &d.Name, &d.ShortName, &d.Description,
		&d.InstanceID, &d.SortOrder, &d.IsActive, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &d, err
}

func (r *orgRepository) ListDepartments(ctx context.Context, companyID uuid.UUID, parentID *uuid.UUID, includeInactive bool, page, count int) ([]model.Department, int, error) {
	offset := (page - 1) * count
	if offset < 0 {
		offset = 0
	}

	whereClause := "WHERE company_id = $1"
	args := []interface{}{companyID}
	argNum := 2

	if parentID != nil {
		whereClause += fmt.Sprintf(" AND parent_department_id = $%d", argNum)
		args = append(args, *parentID)
		argNum++
	}

	if !includeInactive {
		whereClause += " AND is_active = true"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM con_test.departments %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, company_id, parent_department_id, name, short_name, description, instance_id, sort_order, is_active, created_at, updated_at
		FROM con_test.departments %s
		ORDER BY sort_order, name
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, count, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var departments []model.Department
	for rows.Next() {
		var d model.Department
		if err := rows.Scan(&d.ID, &d.CompanyID, &d.ParentDepartmentID, &d.Name, &d.ShortName, &d.Description,
			&d.InstanceID, &d.SortOrder, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, err
		}
		departments = append(departments, d)
	}
	return departments, total, nil
}

func (r *orgRepository) UpdateDepartment(ctx context.Context, d *model.Department) error {
	query := `
		UPDATE con_test.departments
		SET name = $2, short_name = $3, description = $4, sort_order = $5, is_active = $6
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, d.ID, d.Name, d.ShortName, d.Description, d.SortOrder, d.IsActive).
		Scan(&d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *orgRepository) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM con_test.departments WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== Positions ====================

func (r *orgRepository) CreatePosition(ctx context.Context, p *model.Position) error {
	query := `
		INSERT INTO con_test.positions (id, company_id, name, short_name, level, description, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	err := r.pool.QueryRow(ctx, query, p.ID, p.CompanyID, p.Name, p.ShortName, p.Level, p.Description, p.IsActive).
		Scan(&p.CreatedAt, &p.UpdatedAt)
	return err
}

func (r *orgRepository) GetPosition(ctx context.Context, id uuid.UUID) (*model.Position, error) {
	query := `
		SELECT id, company_id, name, short_name, level, description, instance_id, is_active, created_at, updated_at
		FROM con_test.positions WHERE id = $1
	`
	var p model.Position
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.CompanyID, &p.Name, &p.ShortName, &p.Level, &p.Description,
		&p.InstanceID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

func (r *orgRepository) ListPositions(ctx context.Context, companyID uuid.UUID, includeInactive bool, page, count int) ([]model.Position, int, error) {
	offset := (page - 1) * count
	if offset < 0 {
		offset = 0
	}

	whereClause := "WHERE company_id = $1"
	if !includeInactive {
		whereClause += " AND is_active = true"
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM con_test.positions %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, companyID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, company_id, name, short_name, level, description, instance_id, is_active, created_at, updated_at
		FROM con_test.positions %s
		ORDER BY level DESC, name
		LIMIT $2 OFFSET $3
	`, whereClause)

	rows, err := r.pool.Query(ctx, query, companyID, count, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var positions []model.Position
	for rows.Next() {
		var p model.Position
		if err := rows.Scan(&p.ID, &p.CompanyID, &p.Name, &p.ShortName, &p.Level, &p.Description,
			&p.InstanceID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		positions = append(positions, p)
	}
	return positions, total, nil
}

func (r *orgRepository) UpdatePosition(ctx context.Context, p *model.Position) error {
	query := `
		UPDATE con_test.positions
		SET name = $2, short_name = $3, level = $4, description = $5, is_active = $6
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, p.ID, p.Name, p.ShortName, p.Level, p.Description, p.IsActive).
		Scan(&p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *orgRepository) DeletePosition(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM con_test.positions WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== Employees ====================

func (r *orgRepository) CreateEmployee(ctx context.Context, e *model.Employee) error {
	query := `
		INSERT INTO con_test.employees (id, user_id, department_id, position_id, employee_number, hire_date, is_primary, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	err := r.pool.QueryRow(ctx, query, e.ID, e.UserID, e.DepartmentID, e.PositionID, e.EmployeeNumber, e.HireDate, e.IsPrimary, e.IsActive).
		Scan(&e.CreatedAt, &e.UpdatedAt)
	return err
}

func (r *orgRepository) GetEmployee(ctx context.Context, id uuid.UUID) (*model.Employee, error) {
	query := `
		SELECT id, user_id, department_id, position_id, employee_number, hire_date, instance_id, is_primary, is_active, created_at, updated_at
		FROM con_test.employees WHERE id = $1
	`
	var e model.Employee
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.UserID, &e.DepartmentID, &e.PositionID, &e.EmployeeNumber, &e.HireDate,
		&e.InstanceID, &e.IsPrimary, &e.IsActive, &e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &e, err
}

func (r *orgRepository) GetEmployeeByUserID(ctx context.Context, userID uuid.UUID, primaryOnly bool) (*model.Employee, error) {
	query := `
		SELECT id, user_id, department_id, position_id, employee_number, hire_date, instance_id, is_primary, is_active, created_at, updated_at
		FROM con_test.employees
		WHERE user_id = $1 AND is_active = true
	`
	if primaryOnly {
		query += " AND is_primary = true"
	}
	query += " ORDER BY is_primary DESC LIMIT 1"

	var e model.Employee
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&e.ID, &e.UserID, &e.DepartmentID, &e.PositionID, &e.EmployeeNumber, &e.HireDate,
		&e.InstanceID, &e.IsPrimary, &e.IsActive, &e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &e, err
}

func (r *orgRepository) ListEmployees(ctx context.Context, departmentID, companyID *uuid.UUID, includeInactive bool, page, count int) ([]model.Employee, int, error) {
	offset := (page - 1) * count
	if offset < 0 {
		offset = 0
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argNum := 1

	if departmentID != nil {
		whereClause += fmt.Sprintf(" AND e.department_id = $%d", argNum)
		args = append(args, *departmentID)
		argNum++
	}

	if companyID != nil {
		whereClause += fmt.Sprintf(" AND d.company_id = $%d", argNum)
		args = append(args, *companyID)
		argNum++
	}

	if !includeInactive {
		whereClause += " AND e.is_active = true"
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM con_test.employees e
		JOIN con_test.departments d ON e.department_id = d.id
		%s
	`, whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT e.id, e.user_id, e.department_id, e.position_id, e.employee_number, e.hire_date, e.instance_id, e.is_primary, e.is_active, e.created_at, e.updated_at
		FROM con_test.employees e
		JOIN con_test.departments d ON e.department_id = d.id
		%s
		ORDER BY e.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, count, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var employees []model.Employee
	for rows.Next() {
		var e model.Employee
		if err := rows.Scan(&e.ID, &e.UserID, &e.DepartmentID, &e.PositionID, &e.EmployeeNumber, &e.HireDate,
			&e.InstanceID, &e.IsPrimary, &e.IsActive, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		employees = append(employees, e)
	}
	return employees, total, nil
}

func (r *orgRepository) UpdateEmployee(ctx context.Context, e *model.Employee) error {
	query := `
		UPDATE con_test.employees
		SET department_id = $2, position_id = $3, employee_number = $4, hire_date = $5, is_primary = $6, is_active = $7
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.pool.QueryRow(ctx, query, e.ID, e.DepartmentID, e.PositionID, e.EmployeeNumber, e.HireDate, e.IsPrimary, e.IsActive).
		Scan(&e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *orgRepository) DeleteEmployee(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM con_test.employees WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ==================== Enrichment ====================

func (r *orgRepository) GetUserOrgInfo(ctx context.Context, userID uuid.UUID) (*model.UserOrgInfo, error) {
	query := `
		SELECT
			e.user_id,
			c.id as company_id,
			c.name as company_name,
			d.id as department_id,
			d.name as department_name,
			p.id as position_id,
			p.name as position_name,
			c.timezone
		FROM con_test.employees e
		JOIN con_test.departments d ON e.department_id = d.id
		JOIN con_test.companies c ON d.company_id = c.id
		JOIN con_test.positions p ON e.position_id = p.id
		WHERE e.user_id = $1 AND e.is_active = true AND e.is_primary = true
		LIMIT 1
	`

	var info model.UserOrgInfo
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&info.UserID, &info.CompanyID, &info.CompanyName,
		&info.DepartmentID, &info.DepartmentName,
		&info.PositionID, &info.PositionName, &info.Timezone,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return &model.UserOrgInfo{
			UserID:     userID,
			Timezone:   "Europe/Moscow",
			HasOrgData: false,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	info.HasOrgData = true
	return &info, nil
}

func (r *orgRepository) GetUsersOrgInfoBatch(ctx context.Context, userIDs []uuid.UUID) ([]model.UserOrgInfo, error) {
	if len(userIDs) == 0 {
		return []model.UserOrgInfo{}, nil
	}

	query := `
		SELECT
			e.user_id,
			c.id as company_id,
			c.name as company_name,
			d.id as department_id,
			d.name as department_name,
			p.id as position_id,
			p.name as position_name,
			c.timezone
		FROM con_test.employees e
		JOIN con_test.departments d ON e.department_id = d.id
		JOIN con_test.companies c ON d.company_id = c.id
		JOIN con_test.positions p ON e.position_id = p.id
		WHERE e.user_id = ANY($1) AND e.is_active = true AND e.is_primary = true
	`

	rows, err := r.pool.Query(ctx, query, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	infoMap := make(map[uuid.UUID]model.UserOrgInfo)
	for rows.Next() {
		var info model.UserOrgInfo
		if err := rows.Scan(
			&info.UserID, &info.CompanyID, &info.CompanyName,
			&info.DepartmentID, &info.DepartmentName,
			&info.PositionID, &info.PositionName, &info.Timezone,
		); err != nil {
			return nil, err
		}
		info.HasOrgData = true
		infoMap[info.UserID] = info
	}

	// Build result with default values for users without org data
	result := make([]model.UserOrgInfo, len(userIDs))
	for i, uid := range userIDs {
		if info, ok := infoMap[uid]; ok {
			result[i] = info
		} else {
			result[i] = model.UserOrgInfo{
				UserID:     uid,
				Timezone:   "Europe/Moscow",
				HasOrgData: false,
			}
		}
	}
	return result, nil
}
