package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pb "github.com/icegreg/chat-smpl/proto/org"
)

type OrgClient struct {
	conn   *grpc.ClientConn
	client pb.OrgServiceClient
}

func NewOrgClient(addr string) (*OrgClient, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, err
	}

	return &OrgClient{
		conn:   conn,
		client: pb.NewOrgServiceClient(conn),
	}, nil
}

func (c *OrgClient) Close() error {
	return c.conn.Close()
}

// GetUserOrgInfo retrieves organization info for a single user
func (c *OrgClient) GetUserOrgInfo(ctx context.Context, userID string) (*pb.UserOrgInfo, error) {
	return c.client.GetUserOrgInfo(ctx, &pb.GetUserOrgInfoRequest{
		UserId: userID,
	})
}

// GetUsersOrgInfoBatch retrieves organization info for multiple users at once
func (c *OrgClient) GetUsersOrgInfoBatch(ctx context.Context, userIDs []string) ([]*pb.UserOrgInfo, error) {
	resp, err := c.client.GetUsersOrgInfoBatch(ctx, &pb.GetUsersOrgInfoBatchRequest{
		UserIds: userIDs,
	})
	if err != nil {
		return nil, err
	}
	return resp.Users, nil
}

// Company operations

func (c *OrgClient) GetCompany(ctx context.Context, id string) (*pb.Company, error) {
	return c.client.GetCompany(ctx, &pb.GetCompanyRequest{
		Id: id,
	})
}

func (c *OrgClient) ListCompanies(ctx context.Context, parentID string, includeInactive bool, page, count int32) (*pb.ListCompaniesResponse, error) {
	return c.client.ListCompanies(ctx, &pb.ListCompaniesRequest{
		ParentId:        parentID,
		IncludeInactive: includeInactive,
		Page:            page,
		Count:           count,
	})
}

// Department operations

func (c *OrgClient) GetDepartment(ctx context.Context, id string) (*pb.Department, error) {
	return c.client.GetDepartment(ctx, &pb.GetDepartmentRequest{
		Id: id,
	})
}

func (c *OrgClient) ListDepartments(ctx context.Context, companyID, parentDepartmentID string, includeInactive bool, page, count int32) (*pb.ListDepartmentsResponse, error) {
	return c.client.ListDepartments(ctx, &pb.ListDepartmentsRequest{
		CompanyId:          companyID,
		ParentDepartmentId: parentDepartmentID,
		IncludeInactive:    includeInactive,
		Page:               page,
		Count:              count,
	})
}

// Position operations

func (c *OrgClient) GetPosition(ctx context.Context, id string) (*pb.Position, error) {
	return c.client.GetPosition(ctx, &pb.GetPositionRequest{
		Id: id,
	})
}

func (c *OrgClient) ListPositions(ctx context.Context, companyID string, includeInactive bool, page, count int32) (*pb.ListPositionsResponse, error) {
	return c.client.ListPositions(ctx, &pb.ListPositionsRequest{
		CompanyId:       companyID,
		IncludeInactive: includeInactive,
		Page:            page,
		Count:           count,
	})
}

// Employee operations

func (c *OrgClient) GetEmployeeByUserId(ctx context.Context, userID string, primaryOnly bool) (*pb.Employee, error) {
	return c.client.GetEmployeeByUserId(ctx, &pb.GetEmployeeByUserIdRequest{
		UserId:      userID,
		PrimaryOnly: primaryOnly,
	})
}

func (c *OrgClient) ListEmployees(ctx context.Context, departmentID, companyID string, includeInactive bool, page, count int32) (*pb.ListEmployeesResponse, error) {
	return c.client.ListEmployees(ctx, &pb.ListEmployeesRequest{
		DepartmentId:    departmentID,
		CompanyId:       companyID,
		IncludeInactive: includeInactive,
		Page:            page,
		Count:           count,
	})
}
