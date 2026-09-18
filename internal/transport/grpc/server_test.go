package grpc_test

import (
	"context"
	"testing"

	"backend-challenge/internal/pkg/jwt"
	"backend-challenge/internal/repository/mock"
	"backend-challenge/internal/service"
	grpcTransport "backend-challenge/internal/transport/grpc"
	pb "backend-challenge/proto"
)

func setupTestGRPCServer() (*grpcTransport.UserGRPCServer, *service.UserService) {
	repo := mock.NewMockUserRepository()
	tokenManager := jwt.NewTokenManager("test-jwt-secret-key-12345", 1)
	svc := service.NewUserService(repo, tokenManager)
	server := grpcTransport.NewUserGRPCServer(svc)
	return server, svc
}

// TestGRPC_CreateUser ทดสอบการสร้างผู้ใช้ผ่าน gRPC Service
func TestGRPC_CreateUser(t *testing.T) {
	server, _ := setupTestGRPCServer()
	ctx := context.Background()

	res, err := server.CreateUser(ctx, &pb.CreateUserRequest{
		Name:     "GRPC User",
		Email:    "grpc@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected create user to succeed, got %v", err)
	}

	if res.GetId() == "" {
		t.Error("expected non-empty user ID in gRPC response")
	}
	if res.GetName() != "GRPC User" {
		t.Errorf("expected name 'GRPC User', got '%s'", res.GetName())
	}
	if res.GetEmail() != "grpc@example.com" {
		t.Errorf("expected email 'grpc@example.com', got '%s'", res.GetEmail())
	}
}

// TestGRPC_GetUser ทดสอบการดึงข้อมูลผู้ใช้ผ่าน gRPC Service
func TestGRPC_GetUser(t *testing.T) {
	server, _ := setupTestGRPCServer()
	ctx := context.Background()

	// สร้างผู้ใช้ก่อน
	created, err := server.CreateUser(ctx, &pb.CreateUserRequest{
		Name:     "Jane Doe",
		Email:    "jane@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// ค้นหาตาม ID
	found, err := server.GetUser(ctx, &pb.GetUserRequest{
		Id: created.GetId(),
	})
	if err != nil {
		t.Fatalf("expected get user to succeed, got %v", err)
	}

	if found.GetEmail() != "jane@example.com" {
		t.Errorf("expected email 'jane@example.com', got '%s'", found.GetEmail())
	}
}

// TestGRPC_ListUsers ทดสอบการดึงรายชื่อผู้ใช้ทั้งหมดผ่าน gRPC Service
func TestGRPC_ListUsers(t *testing.T) {
	server, _ := setupTestGRPCServer()
	ctx := context.Background()

	_, err := server.CreateUser(ctx, &pb.CreateUserRequest{
		Name:     "User One",
		Email:    "one@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create user one: %v", err)
	}

	_, err = server.CreateUser(ctx, &pb.CreateUserRequest{
		Name:     "User Two",
		Email:    "two@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create user two: %v", err)
	}

	listRes, err := server.ListUsers(ctx, &pb.ListUsersRequest{})
	if err != nil {
		t.Fatalf("expected list users to succeed, got %v", err)
	}

	if len(listRes.GetUsers()) < 2 {
		t.Errorf("expected at least 2 users, got %d", len(listRes.GetUsers()))
	}
}

// TestGRPC_UpdateUser ทดสอบการอัปเดตข้อมูลผู้ใช้ผ่าน gRPC Service
func TestGRPC_UpdateUser(t *testing.T) {
	server, _ := setupTestGRPCServer()
	ctx := context.Background()

	created, err := server.CreateUser(ctx, &pb.CreateUserRequest{
		Name:     "Original Name",
		Email:    "original@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	updated, err := server.UpdateUser(ctx, &pb.UpdateUserRequest{
		Id:    created.GetId(),
		Name:  "Updated Name",
		Email: "updated@example.com",
	})
	if err != nil {
		t.Fatalf("expected update user to succeed, got %v", err)
	}

	if updated.GetName() != "Updated Name" {
		t.Errorf("expected updated name 'Updated Name', got '%s'", updated.GetName())
	}
	if updated.GetEmail() != "updated@example.com" {
		t.Errorf("expected updated email 'updated@example.com', got '%s'", updated.GetEmail())
	}
}

// TestGRPC_DeleteUser ทดสอบการลบผู้ใช้ผ่าน gRPC Service
func TestGRPC_DeleteUser(t *testing.T) {
	server, _ := setupTestGRPCServer()
	ctx := context.Background()

	created, err := server.CreateUser(ctx, &pb.CreateUserRequest{
		Name:     "To Be Deleted",
		Email:    "delete_me@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	delRes, err := server.DeleteUser(ctx, &pb.DeleteUserRequest{
		Id: created.GetId(),
	})
	if err != nil {
		t.Fatalf("expected delete user to succeed, got %v", err)
	}

	if !delRes.GetSuccess() {
		t.Error("expected delete response success to be true")
	}

	// ยืนยันว่าค้นหาไม่เจอแล้ว
	_, err = server.GetUser(ctx, &pb.GetUserRequest{
		Id: created.GetId(),
	})
	if err == nil {
		t.Error("expected get user after delete to return error, got nil")
	}
}
