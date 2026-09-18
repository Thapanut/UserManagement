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

// CreateUser สร้างผู้ใช้ใหม่ผ่าน gRPC
func (s *UserGRPCServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	if req.GetName() == "" || req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "name, email, and password are required")
	}

	user, err := s.userService.CreateUser(ctx, service.CreateUserRequest{
		Name:     req.GetName(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "%s", err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return &pb.UserResponse{
		Id:        user.ID.Hex(),
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
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

	return &pb.UserResponse{
		Id:        user.ID.Hex(),
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}
