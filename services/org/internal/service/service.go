package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/icegreg/chat-smpl/services/org/internal/model"
	"github.com/icegreg/chat-smpl/services/org/internal/repository"
)

type OrgService interface {
	// Companies
	CreateCompany(ctx context.Context, c *model.Company) error
	GetCompany(ctx context.Context, id uuid.UUID) (*model.Company, error)
	ListCompanies(ctx context.Context, parentID *uuid.UUID, includeInactive bool, page, count int) ([]model.Company, int, error)
	UpdateCompany(ctx context.Context, c *model.Company) error
	DeleteCompany(ctx context.Context, id uuid.UUID) error
	GetCompanyHierarchy(ctx context.Context, id uuid.UUID) (*model.CompanyHierarchy, error)

	// Departments
	CreateDepartment(ctx context.Context, d *model.Department) error
	GetDepartment(ctx context.Context, id uuid.UUID) (*model.Department, error)
	ListDepartments(ctx context.Context, companyID uuid.UUID, parentID *uuid.UUID, includeInactive bool, page, count int) ([]model.Department, int, error)
	UpdateDepartment(ctx context.Context, d *model.Department) error
	DeleteDepartment(ctx context.Context, id uuid.UUID) error
	GetDepartmentHierarchy(ctx context.Context, id uuid.UUID) (*model.DepartmentHierarchy, error)

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

type orgService struct {
	repo repository.OrgRepository
}

func NewOrgService(repo repository.OrgRepository) OrgService {
	return &orgService{repo: repo}
}

// ==================== Companies ====================

func (s *orgService) CreateCompany(ctx context.Context, c *model.Company) error {
	if c.Timezone == "" {
		c.Timezone = "Europe/Moscow"
	}
	c.IsActive = true
	return s.repo.CreateCompany(ctx, c)
}

func (s *orgService) GetCompany(ctx context.Context, id uuid.UUID) (*model.Company, error) {
	return s.repo.GetCompany(ctx, id)
}

func (s *orgService) ListCompanies(ctx context.Context, parentID *uuid.UUID, includeInactive bool, page, count int) ([]model.Company, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}
	if count > 100 {
		count = 100
	}
	return s.repo.ListCompanies(ctx, parentID, includeInactive, page, count)
}

func (s *orgService) UpdateCompany(ctx context.Context, c *model.Company) error {
	return s.repo.UpdateCompany(ctx, c)
}

func (s *orgService) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCompany(ctx, id)
}

func (s *orgService) GetCompanyHierarchy(ctx context.Context, id uuid.UUID) (*model.CompanyHierarchy, error) {
	company, err := s.repo.GetCompany(ctx, id)
	if err != nil {
		return nil, err
	}

	hierarchy := &model.CompanyHierarchy{
		Company:  company,
		Children: []*model.CompanyHierarchy{},
	}

	// Get children
	children, _, err := s.repo.ListCompanies(ctx, &id, false, 1, 100)
	if err != nil {
		return nil, err
	}

	for _, child := range children {
		childHierarchy, err := s.GetCompanyHierarchy(ctx, child.ID)
		if err != nil {
			return nil, err
		}
		hierarchy.Children = append(hierarchy.Children, childHierarchy)
	}

	return hierarchy, nil
}

// ==================== Departments ====================

func (s *orgService) CreateDepartment(ctx context.Context, d *model.Department) error {
	d.IsActive = true
	return s.repo.CreateDepartment(ctx, d)
}

func (s *orgService) GetDepartment(ctx context.Context, id uuid.UUID) (*model.Department, error) {
	return s.repo.GetDepartment(ctx, id)
}

func (s *orgService) ListDepartments(ctx context.Context, companyID uuid.UUID, parentID *uuid.UUID, includeInactive bool, page, count int) ([]model.Department, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}
	if count > 100 {
		count = 100
	}
	return s.repo.ListDepartments(ctx, companyID, parentID, includeInactive, page, count)
}

func (s *orgService) UpdateDepartment(ctx context.Context, d *model.Department) error {
	return s.repo.UpdateDepartment(ctx, d)
}

func (s *orgService) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteDepartment(ctx, id)
}

func (s *orgService) GetDepartmentHierarchy(ctx context.Context, id uuid.UUID) (*model.DepartmentHierarchy, error) {
	dept, err := s.repo.GetDepartment(ctx, id)
	if err != nil {
		return nil, err
	}

	hierarchy := &model.DepartmentHierarchy{
		Department: dept,
		Children:   []*model.DepartmentHierarchy{},
	}

	// Get children
	children, _, err := s.repo.ListDepartments(ctx, dept.CompanyID, &id, false, 1, 100)
	if err != nil {
		return nil, err
	}

	for _, child := range children {
		childHierarchy, err := s.GetDepartmentHierarchy(ctx, child.ID)
		if err != nil {
			return nil, err
		}
		hierarchy.Children = append(hierarchy.Children, childHierarchy)
	}

	return hierarchy, nil
}

// ==================== Positions ====================

func (s *orgService) CreatePosition(ctx context.Context, p *model.Position) error {
	p.IsActive = true
	return s.repo.CreatePosition(ctx, p)
}

func (s *orgService) GetPosition(ctx context.Context, id uuid.UUID) (*model.Position, error) {
	return s.repo.GetPosition(ctx, id)
}

func (s *orgService) ListPositions(ctx context.Context, companyID uuid.UUID, includeInactive bool, page, count int) ([]model.Position, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}
	if count > 100 {
		count = 100
	}
	return s.repo.ListPositions(ctx, companyID, includeInactive, page, count)
}

func (s *orgService) UpdatePosition(ctx context.Context, p *model.Position) error {
	return s.repo.UpdatePosition(ctx, p)
}

func (s *orgService) DeletePosition(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePosition(ctx, id)
}

// ==================== Employees ====================

func (s *orgService) CreateEmployee(ctx context.Context, e *model.Employee) error {
	e.IsActive = true
	return s.repo.CreateEmployee(ctx, e)
}

func (s *orgService) GetEmployee(ctx context.Context, id uuid.UUID) (*model.Employee, error) {
	emp, err := s.repo.GetEmployee(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.enrichEmployee(ctx, emp)
}

func (s *orgService) GetEmployeeByUserID(ctx context.Context, userID uuid.UUID, primaryOnly bool) (*model.Employee, error) {
	emp, err := s.repo.GetEmployeeByUserID(ctx, userID, primaryOnly)
	if err != nil {
		return nil, err
	}
	return s.enrichEmployee(ctx, emp)
}

func (s *orgService) enrichEmployee(ctx context.Context, emp *model.Employee) (*model.Employee, error) {
	// Get department
	dept, err := s.repo.GetDepartment(ctx, emp.DepartmentID)
	if err == nil {
		emp.Department = dept

		// Get company
		company, err := s.repo.GetCompany(ctx, dept.CompanyID)
		if err == nil {
			emp.Company = company
		}
	}

	// Get position
	pos, err := s.repo.GetPosition(ctx, emp.PositionID)
	if err == nil {
		emp.Position = pos
	}

	return emp, nil
}

func (s *orgService) ListEmployees(ctx context.Context, departmentID, companyID *uuid.UUID, includeInactive bool, page, count int) ([]model.Employee, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}
	if count > 100 {
		count = 100
	}
	return s.repo.ListEmployees(ctx, departmentID, companyID, includeInactive, page, count)
}

func (s *orgService) UpdateEmployee(ctx context.Context, e *model.Employee) error {
	return s.repo.UpdateEmployee(ctx, e)
}

func (s *orgService) DeleteEmployee(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteEmployee(ctx, id)
}

// ==================== Enrichment ====================

func (s *orgService) GetUserOrgInfo(ctx context.Context, userID uuid.UUID) (*model.UserOrgInfo, error) {
	return s.repo.GetUserOrgInfo(ctx, userID)
}

func (s *orgService) GetUsersOrgInfoBatch(ctx context.Context, userIDs []uuid.UUID) ([]model.UserOrgInfo, error) {
	return s.repo.GetUsersOrgInfoBatch(ctx, userIDs)
}
