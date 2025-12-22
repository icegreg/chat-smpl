package model

import (
	"time"

	"github.com/google/uuid"
)

// Company represents an organization/company
type Company struct {
	ID          uuid.UUID  `db:"id"`
	ParentID    *uuid.UUID `db:"parent_id"`
	Name        string     `db:"name"`
	ShortName   *string    `db:"short_name"`
	Description *string    `db:"description"`
	InstanceID  *uuid.UUID `db:"instance_id"`
	Timezone    string     `db:"timezone"`
	IsActive    bool       `db:"is_active"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// Department represents a structural unit within a company
type Department struct {
	ID                 uuid.UUID  `db:"id"`
	CompanyID          uuid.UUID  `db:"company_id"`
	ParentDepartmentID *uuid.UUID `db:"parent_department_id"`
	Name               string     `db:"name"`
	ShortName          *string    `db:"short_name"`
	Description        *string    `db:"description"`
	InstanceID         *uuid.UUID `db:"instance_id"`
	SortOrder          int        `db:"sort_order"`
	IsActive           bool       `db:"is_active"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
}

// Position represents a job position within a company
type Position struct {
	ID          uuid.UUID  `db:"id"`
	CompanyID   uuid.UUID  `db:"company_id"`
	Name        string     `db:"name"`
	ShortName   *string    `db:"short_name"`
	Level       int        `db:"level"`
	Description *string    `db:"description"`
	InstanceID  *uuid.UUID `db:"instance_id"`
	IsActive    bool       `db:"is_active"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// Employee represents a user's employment record
type Employee struct {
	ID             uuid.UUID  `db:"id"`
	UserID         uuid.UUID  `db:"user_id"`
	DepartmentID   uuid.UUID  `db:"department_id"`
	PositionID     uuid.UUID  `db:"position_id"`
	EmployeeNumber *string    `db:"employee_number"`
	HireDate       *time.Time `db:"hire_date"`
	InstanceID     *uuid.UUID `db:"instance_id"`
	IsPrimary      bool       `db:"is_primary"`
	IsActive       bool       `db:"is_active"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`

	// Expanded fields (populated by service layer)
	Department *Department `db:"-"`
	Position   *Position   `db:"-"`
	Company    *Company    `db:"-"`
}

// UserOrgInfo contains organization info for user profile enrichment
type UserOrgInfo struct {
	UserID         uuid.UUID
	CompanyID      uuid.UUID
	CompanyName    string
	DepartmentID   uuid.UUID
	DepartmentName string
	PositionID     uuid.UUID
	PositionName   string
	Timezone       string
	HasOrgData     bool
}

// CompanyHierarchy represents a company with its children
type CompanyHierarchy struct {
	Company  *Company
	Children []*CompanyHierarchy
}

// DepartmentHierarchy represents a department with its children
type DepartmentHierarchy struct {
	Department *Department
	Children   []*DepartmentHierarchy
}
