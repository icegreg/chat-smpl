package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/icegreg/chat-smpl/proto/org"
	"github.com/icegreg/chat-smpl/services/org/internal/model"
	"github.com/icegreg/chat-smpl/services/org/internal/repository"
	"github.com/icegreg/chat-smpl/services/org/internal/service"
)

type OrgServer struct {
	pb.UnimplementedOrgServiceServer
	service service.OrgService
}

func NewOrgServer(svc service.OrgService) *OrgServer {
	return &OrgServer{service: svc}
}

// ==================== Companies ====================

func (s *OrgServer) CreateCompany(ctx context.Context, req *pb.CreateCompanyRequest) (*pb.Company, error) {
	c := &model.Company{
		Name:        req.Name,
		ShortName:   strPtr(req.ShortName),
		Description: strPtr(req.Description),
		Timezone:    req.Timezone,
	}
	if req.ParentId != "" {
		parentID, err := uuid.Parse(req.ParentId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_id")
		}
		c.ParentID = &parentID
	}

	if err := s.service.CreateCompany(ctx, c); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return companyToProto(c), nil
}

func (s *OrgServer) GetCompany(ctx context.Context, req *pb.GetCompanyRequest) (*pb.Company, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	c, err := s.service.GetCompany(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "company not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return companyToProto(c), nil
}

func (s *OrgServer) ListCompanies(ctx context.Context, req *pb.ListCompaniesRequest) (*pb.ListCompaniesResponse, error) {
	var parentID *uuid.UUID
	if req.ParentId != "" {
		pid, err := uuid.Parse(req.ParentId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_id")
		}
		parentID = &pid
	}

	companies, total, err := s.service.ListCompanies(ctx, parentID, req.IncludeInactive, int(req.Page), int(req.Count))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbCompanies := make([]*pb.Company, len(companies))
	for i, c := range companies {
		pbCompanies[i] = companyToProto(&c)
	}

	return &pb.ListCompaniesResponse{
		Companies:  pbCompanies,
		Pagination: makePagination(int(req.Page), int(req.Count), total),
	}, nil
}

func (s *OrgServer) UpdateCompany(ctx context.Context, req *pb.UpdateCompanyRequest) (*pb.Company, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	c := &model.Company{
		ID:          id,
		Name:        req.Name,
		ShortName:   strPtr(req.ShortName),
		Description: strPtr(req.Description),
		Timezone:    req.Timezone,
		IsActive:    req.IsActive,
	}

	if err := s.service.UpdateCompany(ctx, c); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "company not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return companyToProto(c), nil
}

func (s *OrgServer) DeleteCompany(ctx context.Context, req *pb.DeleteCompanyRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	if err := s.service.DeleteCompany(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "company not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *OrgServer) GetCompanyHierarchy(ctx context.Context, req *pb.GetCompanyRequest) (*pb.CompanyHierarchy, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	h, err := s.service.GetCompanyHierarchy(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "company not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return companyHierarchyToProto(h), nil
}

// ==================== Departments ====================

func (s *OrgServer) CreateDepartment(ctx context.Context, req *pb.CreateDepartmentRequest) (*pb.Department, error) {
	companyID, err := uuid.Parse(req.CompanyId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	d := &model.Department{
		CompanyID:   companyID,
		Name:        req.Name,
		ShortName:   strPtr(req.ShortName),
		Description: strPtr(req.Description),
		SortOrder:   int(req.SortOrder),
	}
	if req.ParentDepartmentId != "" {
		pid, err := uuid.Parse(req.ParentDepartmentId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_department_id")
		}
		d.ParentDepartmentID = &pid
	}

	if err := s.service.CreateDepartment(ctx, d); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return departmentToProto(d), nil
}

func (s *OrgServer) GetDepartment(ctx context.Context, req *pb.GetDepartmentRequest) (*pb.Department, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	d, err := s.service.GetDepartment(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "department not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return departmentToProto(d), nil
}

func (s *OrgServer) ListDepartments(ctx context.Context, req *pb.ListDepartmentsRequest) (*pb.ListDepartmentsResponse, error) {
	companyID, err := uuid.Parse(req.CompanyId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	var parentID *uuid.UUID
	if req.ParentDepartmentId != "" {
		pid, err := uuid.Parse(req.ParentDepartmentId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_department_id")
		}
		parentID = &pid
	}

	departments, total, err := s.service.ListDepartments(ctx, companyID, parentID, req.IncludeInactive, int(req.Page), int(req.Count))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbDepartments := make([]*pb.Department, len(departments))
	for i, d := range departments {
		pbDepartments[i] = departmentToProto(&d)
	}

	return &pb.ListDepartmentsResponse{
		Departments: pbDepartments,
		Pagination:  makePagination(int(req.Page), int(req.Count), total),
	}, nil
}

func (s *OrgServer) UpdateDepartment(ctx context.Context, req *pb.UpdateDepartmentRequest) (*pb.Department, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	d := &model.Department{
		ID:          id,
		Name:        req.Name,
		ShortName:   strPtr(req.ShortName),
		Description: strPtr(req.Description),
		SortOrder:   int(req.SortOrder),
		IsActive:    req.IsActive,
	}

	if err := s.service.UpdateDepartment(ctx, d); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "department not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return departmentToProto(d), nil
}

func (s *OrgServer) DeleteDepartment(ctx context.Context, req *pb.DeleteDepartmentRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	if err := s.service.DeleteDepartment(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "department not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *OrgServer) GetDepartmentHierarchy(ctx context.Context, req *pb.GetDepartmentRequest) (*pb.DepartmentHierarchy, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	h, err := s.service.GetDepartmentHierarchy(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "department not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return departmentHierarchyToProto(h), nil
}

// ==================== Positions ====================

func (s *OrgServer) CreatePosition(ctx context.Context, req *pb.CreatePositionRequest) (*pb.Position, error) {
	companyID, err := uuid.Parse(req.CompanyId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	p := &model.Position{
		CompanyID:   companyID,
		Name:        req.Name,
		ShortName:   strPtr(req.ShortName),
		Level:       int(req.Level),
		Description: strPtr(req.Description),
	}

	if err := s.service.CreatePosition(ctx, p); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return positionToProto(p), nil
}

func (s *OrgServer) GetPosition(ctx context.Context, req *pb.GetPositionRequest) (*pb.Position, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	p, err := s.service.GetPosition(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "position not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return positionToProto(p), nil
}

func (s *OrgServer) ListPositions(ctx context.Context, req *pb.ListPositionsRequest) (*pb.ListPositionsResponse, error) {
	companyID, err := uuid.Parse(req.CompanyId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	positions, total, err := s.service.ListPositions(ctx, companyID, req.IncludeInactive, int(req.Page), int(req.Count))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbPositions := make([]*pb.Position, len(positions))
	for i, p := range positions {
		pbPositions[i] = positionToProto(&p)
	}

	return &pb.ListPositionsResponse{
		Positions:  pbPositions,
		Pagination: makePagination(int(req.Page), int(req.Count), total),
	}, nil
}

func (s *OrgServer) UpdatePosition(ctx context.Context, req *pb.UpdatePositionRequest) (*pb.Position, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	p := &model.Position{
		ID:          id,
		Name:        req.Name,
		ShortName:   strPtr(req.ShortName),
		Level:       int(req.Level),
		Description: strPtr(req.Description),
		IsActive:    req.IsActive,
	}

	if err := s.service.UpdatePosition(ctx, p); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "position not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return positionToProto(p), nil
}

func (s *OrgServer) DeletePosition(ctx context.Context, req *pb.DeletePositionRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	if err := s.service.DeletePosition(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "position not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// ==================== Employees ====================

func (s *OrgServer) CreateEmployee(ctx context.Context, req *pb.CreateEmployeeRequest) (*pb.Employee, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	deptID, err := uuid.Parse(req.DepartmentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid department_id")
	}
	posID, err := uuid.Parse(req.PositionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid position_id")
	}

	e := &model.Employee{
		UserID:         userID,
		DepartmentID:   deptID,
		PositionID:     posID,
		EmployeeNumber: strPtr(req.EmployeeNumber),
		IsPrimary:      req.IsPrimary,
	}

	if err := s.service.CreateEmployee(ctx, e); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Fetch enriched employee
	enriched, _ := s.service.GetEmployee(ctx, e.ID)
	if enriched != nil {
		return employeeToProto(enriched), nil
	}
	return employeeToProto(e), nil
}

func (s *OrgServer) GetEmployee(ctx context.Context, req *pb.GetEmployeeRequest) (*pb.Employee, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	e, err := s.service.GetEmployee(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "employee not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return employeeToProto(e), nil
}

func (s *OrgServer) GetEmployeeByUserId(ctx context.Context, req *pb.GetEmployeeByUserIdRequest) (*pb.Employee, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	e, err := s.service.GetEmployeeByUserID(ctx, userID, req.PrimaryOnly)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "employee not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return employeeToProto(e), nil
}

func (s *OrgServer) ListEmployees(ctx context.Context, req *pb.ListEmployeesRequest) (*pb.ListEmployeesResponse, error) {
	var deptID, companyID *uuid.UUID
	if req.DepartmentId != "" {
		id, err := uuid.Parse(req.DepartmentId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid department_id")
		}
		deptID = &id
	}
	if req.CompanyId != "" {
		id, err := uuid.Parse(req.CompanyId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid company_id")
		}
		companyID = &id
	}

	employees, total, err := s.service.ListEmployees(ctx, deptID, companyID, req.IncludeInactive, int(req.Page), int(req.Count))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbEmployees := make([]*pb.Employee, len(employees))
	for i, e := range employees {
		pbEmployees[i] = employeeToProto(&e)
	}

	return &pb.ListEmployeesResponse{
		Employees:  pbEmployees,
		Pagination: makePagination(int(req.Page), int(req.Count), total),
	}, nil
}

func (s *OrgServer) UpdateEmployee(ctx context.Context, req *pb.UpdateEmployeeRequest) (*pb.Employee, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	deptID, err := uuid.Parse(req.DepartmentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid department_id")
	}
	posID, err := uuid.Parse(req.PositionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid position_id")
	}

	e := &model.Employee{
		ID:             id,
		DepartmentID:   deptID,
		PositionID:     posID,
		EmployeeNumber: strPtr(req.EmployeeNumber),
		IsPrimary:      req.IsPrimary,
		IsActive:       req.IsActive,
	}

	if err := s.service.UpdateEmployee(ctx, e); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "employee not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return employeeToProto(e), nil
}

func (s *OrgServer) DeleteEmployee(ctx context.Context, req *pb.DeleteEmployeeRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	if err := s.service.DeleteEmployee(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "employee not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &emptypb.Empty{}, nil
}

// ==================== Enrichment ====================

func (s *OrgServer) GetUserOrgInfo(ctx context.Context, req *pb.GetUserOrgInfoRequest) (*pb.UserOrgInfo, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	info, err := s.service.GetUserOrgInfo(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return userOrgInfoToProto(info), nil
}

func (s *OrgServer) GetUsersOrgInfoBatch(ctx context.Context, req *pb.GetUsersOrgInfoBatchRequest) (*pb.GetUsersOrgInfoBatchResponse, error) {
	userIDs := make([]uuid.UUID, len(req.UserIds))
	for i, id := range req.UserIds {
		uid, err := uuid.Parse(id)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user_id")
		}
		userIDs[i] = uid
	}

	infos, err := s.service.GetUsersOrgInfoBatch(ctx, userIDs)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pbInfos := make([]*pb.UserOrgInfo, len(infos))
	for i, info := range infos {
		pbInfos[i] = userOrgInfoToProto(&info)
	}

	return &pb.GetUsersOrgInfoBatchResponse{Users: pbInfos}, nil
}

// ==================== Helpers ====================

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func uuidStr(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func makePagination(page, count, total int) *pb.Pagination {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}
	totalPages := (total + count - 1) / count
	return &pb.Pagination{
		Page:       int32(page),
		Count:      int32(count),
		Total:      int32(total),
		TotalPages: int32(totalPages),
	}
}

func companyToProto(c *model.Company) *pb.Company {
	return &pb.Company{
		Id:          c.ID.String(),
		ParentId:    uuidStr(c.ParentID),
		Name:        c.Name,
		ShortName:   strVal(c.ShortName),
		Description: strVal(c.Description),
		Timezone:    c.Timezone,
		InstanceId:  uuidStr(c.InstanceID),
		IsActive:    c.IsActive,
		CreatedAt:   timestamppb.New(c.CreatedAt),
		UpdatedAt:   timestamppb.New(c.UpdatedAt),
	}
}

func companyHierarchyToProto(h *model.CompanyHierarchy) *pb.CompanyHierarchy {
	children := make([]*pb.CompanyHierarchy, len(h.Children))
	for i, child := range h.Children {
		children[i] = companyHierarchyToProto(child)
	}
	return &pb.CompanyHierarchy{
		Company:  companyToProto(h.Company),
		Children: children,
	}
}

func departmentToProto(d *model.Department) *pb.Department {
	return &pb.Department{
		Id:                 d.ID.String(),
		CompanyId:          d.CompanyID.String(),
		ParentDepartmentId: uuidStr(d.ParentDepartmentID),
		Name:               d.Name,
		ShortName:          strVal(d.ShortName),
		Description:        strVal(d.Description),
		SortOrder:          int32(d.SortOrder),
		InstanceId:         uuidStr(d.InstanceID),
		IsActive:           d.IsActive,
		CreatedAt:          timestamppb.New(d.CreatedAt),
		UpdatedAt:          timestamppb.New(d.UpdatedAt),
	}
}

func departmentHierarchyToProto(h *model.DepartmentHierarchy) *pb.DepartmentHierarchy {
	children := make([]*pb.DepartmentHierarchy, len(h.Children))
	for i, child := range h.Children {
		children[i] = departmentHierarchyToProto(child)
	}
	return &pb.DepartmentHierarchy{
		Department: departmentToProto(h.Department),
		Children:   children,
	}
}

func positionToProto(p *model.Position) *pb.Position {
	return &pb.Position{
		Id:          p.ID.String(),
		CompanyId:   p.CompanyID.String(),
		Name:        p.Name,
		ShortName:   strVal(p.ShortName),
		Level:       int32(p.Level),
		Description: strVal(p.Description),
		InstanceId:  uuidStr(p.InstanceID),
		IsActive:    p.IsActive,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}

func employeeToProto(e *model.Employee) *pb.Employee {
	pb := &pb.Employee{
		Id:             e.ID.String(),
		UserId:         e.UserID.String(),
		DepartmentId:   e.DepartmentID.String(),
		PositionId:     e.PositionID.String(),
		EmployeeNumber: strVal(e.EmployeeNumber),
		InstanceId:     uuidStr(e.InstanceID),
		IsPrimary:      e.IsPrimary,
		IsActive:       e.IsActive,
		CreatedAt:      timestamppb.New(e.CreatedAt),
		UpdatedAt:      timestamppb.New(e.UpdatedAt),
	}
	if e.HireDate != nil {
		pb.HireDate = e.HireDate.Format("2006-01-02")
	}
	if e.Department != nil {
		pb.Department = departmentToProto(e.Department)
	}
	if e.Position != nil {
		pb.Position = positionToProto(e.Position)
	}
	if e.Company != nil {
		pb.Company = companyToProto(e.Company)
	}
	return pb
}

func userOrgInfoToProto(info *model.UserOrgInfo) *pb.UserOrgInfo {
	return &pb.UserOrgInfo{
		UserId:         info.UserID.String(),
		CompanyId:      info.CompanyID.String(),
		CompanyName:    info.CompanyName,
		DepartmentId:   info.DepartmentID.String(),
		DepartmentName: info.DepartmentName,
		PositionId:     info.PositionID.String(),
		PositionName:   info.PositionName,
		Timezone:       info.Timezone,
		HasOrgData:     info.HasOrgData,
	}
}
