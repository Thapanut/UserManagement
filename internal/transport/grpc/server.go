package grpc

import (
	"context"
	"errors"
	"time"

	"backend-challenge/internal/domain"
	"backend-challenge/internal/service"
	pb "backend-challenge/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserGRPCServer ตัวประมวลผล gRPC service สำหรับ User
type UserGRPCServer struct {
	pb.UnimplementedUserServiceServer
	userService *service.UserService
}

// NewUserGRPCServer สร้าง Instance ใหม่ของ UserGRPCServer
func NewUserGRPCServer(userService *service.UserService) *UserGRPCServer {
	return &UserGRPCServer{
		userService: userService,
	}
}

// CreateUser สร้างผู้ใช้ใหม่ผ่าน gRPC (รับ gRPC DTO -> แปลงส่งให้ Service -> แปลง Entity กลับเป็น Response DTO)
func (s *UserGRPCServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	if req.GetName() == "" || req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name, email, and password are required")
	}

	serviceReq := toCreateUserServiceRequest(req)
	user, err := s.userService.CreateUser(ctx, serviceReq)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "%s", err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return toUserResponse(user), nil
}

// GetUser ดึงข้อมูลผู้ใช้ตาม ID ผ่าน gRPC
func (s *UserGRPCServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user id is required")
	}

	user, err := s.userService.GetUserByID(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "%s", err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}

	return toUserResponse(user), nil
}

// ListUsers ดึงรายชื่อผู้ใช้ทั้งหมดผ่าน gRPC
func (s *UserGRPCServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	users, err := s.userService.ListUsers(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	var pbUsers []*pb.UserResponse
	for _, u := range users {
		pbUsers = append(pbUsers, toUserResponse(u))
	}

	return &pb.ListUsersResponse{
		Users: pbUsers,
	}, nil
}

// UpdateUser ปรับปรุงข้อมูลผู้ใช้ผ่าน gRPC (รองรับ Partial Update: ชื่อ และ/หรือ อีเมล)
func (s *UserGRPCServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user id is required")
	}

	if req.GetName() == "" && req.GetEmail() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "at least one field (name or email) must be provided for update")
	}

	serviceReq := toUpdateUserServiceRequest(req)
	user, err := s.userService.UpdateUser(ctx, req.GetId(), serviceReq)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "%s", err.Error())
		}
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "%s", err.Error())
		}
		return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
	}

	return toUserResponse(user), nil
}

// DeleteUser ลบผู้ใช้ตาม ID ผ่าน gRPC
func (s *UserGRPCServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user id is required")
	}

	err := s.userService.DeleteUser(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, domain.ErrInvalidID) {
			return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "%s", err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}

	return &pb.DeleteUserResponse{
		Success: true,
		Message: "user deleted successfully",
	}, nil
}

// ==========================================
// DTO <-> Entity Mappers (Separation of Concerns)
// ==========================================

// toCreateUserServiceRequest แปลง Protobuf Request DTO เป็น Service Request DTO
func toCreateUserServiceRequest(req *pb.CreateUserRequest) service.CreateUserRequest {
	return service.CreateUserRequest{
		Name:     req.GetName(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}
}

// toUpdateUserServiceRequest แปลง Protobuf Update Request DTO เป็น Service Request DTO
func toUpdateUserServiceRequest(req *pb.UpdateUserRequest) service.UpdateUserRequest {
	return service.UpdateUserRequest{
		Name:  req.GetName(),
		Email: req.GetEmail(),
	}
}

// toUserResponse แปลง Domain Entity เป็น Protobuf Response DTO สำหรับส่งข้ามเครือข่าย
func toUserResponse(user *domain.User) *pb.UserResponse {
	if user == nil {
		return nil
	}
	return &pb.UserResponse{
		Id:        user.ID.Hex(),
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}
}
