package service_test

import (
	"context"
	"errors"
	"testing"

	"backend-challenge/internal/domain"
	"backend-challenge/internal/pkg/jwt"
	"backend-challenge/internal/repository/mock"
	"backend-challenge/internal/service"
)

// setupTestEnv สร้าง environment สำหรับการทดสอบ (MockRepo + UserService)
func setupTestEnv() (*mock.MockUserRepository, *service.UserService) {
	repo := mock.NewMockUserRepository()
	tokenManager := jwt.NewTokenManager("test-secret-key-12345", 1)
	svc := service.NewUserService(repo, tokenManager)
	return repo, svc
}

// TestRegister_Success ทดสอบการลงทะเบียนสำเร็จ
func TestRegister_Success(t *testing.T) {
	_, svc := setupTestEnv()
	ctx := context.Background()

	req := service.RegisterRequest{
		Name:     "Alice Wonderland",
		Email:    "alice@example.com",
		Password: "password123",
	}

	user, err := svc.Register(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID.IsZero() {
		t.Errorf("expected auto-generated ID, got zero")
	}
	if user.Name != req.Name {
		t.Errorf("expected name %s, got %s", req.Name, user.Name)
	}
	if user.Email != req.Email {
		t.Errorf("expected email %s, got %s", req.Email, user.Email)
	}
	if user.Password == "" || user.Password == req.Password {
		t.Errorf("expected password to be hashed, not plaintext")
	}
}

// TestRegister_DuplicateEmail ทดสอบการปฏิเสธเมื่ออีเมลซ้ำ
func TestRegister_DuplicateEmail(t *testing.T) {
	_, svc := setupTestEnv()
	ctx := context.Background()

	req := service.RegisterRequest{
		Name:     "Alice Wonderland",
		Email:    "alice@example.com",
		Password: "password123",
	}

	_, err := svc.Register(ctx, req)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// พยายามลงทะเบียนซ้ำด้วยอีเมลเดิม
	_, err = svc.Register(ctx, req)
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

// TestRegister_ValidationErrors ทดสอบการตรวจสอบความถูกต้องของข้อมูล (Validation)
func TestRegister_ValidationErrors(t *testing.T) {
	_, svc := setupTestEnv()
	ctx := context.Background()

	tests := []struct {
		name        string
		req         service.RegisterRequest
		expectedErr string
	}{
		{
			name:        "Empty name",
			req:         service.RegisterRequest{Name: "", Email: "test@example.com", Password: "password123"},
			expectedErr: "name is required",
		},
		{
			name:        "Invalid email",
			req:         service.RegisterRequest{Name: "Bob", Email: "invalid-email", Password: "password123"},
			expectedErr: "invalid email address format",
		},
		{
			name:        "Short password",
			req:         service.RegisterRequest{Name: "Bob", Email: "bob@example.com", Password: "123"},
			expectedErr: "password must be at least 6 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(ctx, tt.req)
			if err == nil || err.Error() != tt.expectedErr {
				t.Fatalf("expected error '%s', got '%v'", tt.expectedErr, err)
			}
		})
	}
}

// TestLogin_Success ทดสอบการเข้าสู่ระบบสำเร็จและได้ JWT Token
func TestLogin_Success(t *testing.T) {
	_, svc := setupTestEnv()
	ctx := context.Background()

	// ลงทะเบียนผู้ใช้ก่อน
	_, err := svc.Register(ctx, service.RegisterRequest{
		Name:     "Bob Builder",
		Email:    "bob@example.com",
		Password: "secretpassword",
	})
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	// ทดสอบ Login
	loginRes, err := svc.Login(ctx, service.LoginRequest{
		Email:    "bob@example.com",
		Password: "secretpassword",
	})
	if err != nil {
		t.Fatalf("expected login success, got %v", err)
	}

	if loginRes.Token == "" {
		t.Errorf("expected JWT token string, got empty")
	}
	if loginRes.User.Email != "bob@example.com" {
		t.Errorf("expected email bob@example.com, got %s", loginRes.User.Email)
	}
}

// TestLogin_InvalidCredentials ทดสอบกรณีรหัสผ่านหรืออีเมลผิด
func TestLogin_InvalidCredentials(t *testing.T) {
	_, svc := setupTestEnv()
	ctx := context.Background()

	// ลงทะเบียนผู้ใช้
	_, err := svc.Register(ctx, service.RegisterRequest{
		Name:     "Charlie",
		Email:    "charlie@example.com",
		Password: "correctpassword",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// 1. รหัสผ่านผิด
	_, err = svc.Login(ctx, service.LoginRequest{
		Email:    "charlie@example.com",
		Password: "wrongpassword",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for wrong password, got %v", err)
	}

	// 2. ไม่พบอีเมล
	_, err = svc.Login(ctx, service.LoginRequest{
		Email:    "unknown@example.com",
		Password: "anypassword",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for unknown user, got %v", err)
	}
}

// TestGetUserByID ทดสอบการดึงข้อมูลผู้ใช้จาก ID
func TestGetUserByID(t *testing.T) {
	_, svc := setupTestEnv()
	ctx := context.Background()

	created, err := svc.Register(ctx, service.RegisterRequest{
		Name:     "David",
		Email:    "david@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// ค้นหาด้วย ID ที่ถูกต้อง
	found, err := svc.GetUserByID(ctx, created.ID.Hex())
	if err != nil {
		t.Fatalf("expected to find user, got error: %v", err)
	}
	if found.Email != "david@example.com" {
		t.Errorf("expected email david@example.com, got %s", found.Email)
	}

	// ค้นหาด้วย Invalid ObjectID Format
	_, err = svc.GetUserByID(ctx, "invalid-hex-id")
	if !errors.Is(err, domain.ErrInvalidID) {
		t.Errorf("expected ErrInvalidID, got %v", err)
	}
}

// TestListUsers ทดสอบการดึงรายชื่อผู้ใช้ทั้งหมด
func TestListUsers(t *testing.T) {
	_, svc := setupTestEnv()
	ctx := context.Background()

	// ก่อนสร้าง ต้องได้รายการว่าง
	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}

	// สร้าง 2 คน
	svc.Register(ctx, service.RegisterRequest{Name: "User 1", Email: "u1@example.com", Password: "password123"})
	svc.Register(ctx, service.RegisterRequest{Name: "User 2", Email: "u2@example.com", Password: "password123"})

	users, err = svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

// TestUpdateUser ทดสอบการอัปเดตชื่อและอีเมล
func TestUpdateUser(t *testing.T) {
	_, svc := setupTestEnv()
	ctx := context.Background()

	created, err := svc.Register(ctx, service.RegisterRequest{
		Name:     "Old Name",
		Email:    "old@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// อัปเดตข้อมูลสำเร็จ
	updated, err := svc.UpdateUser(ctx, created.ID.Hex(), service.UpdateUserRequest{
		Name:  "New Name",
		Email: "new@example.com",
	})
	if err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}
	if updated.Name != "New Name" || updated.Email != "new@example.com" {
		t.Errorf("updated fields mismatch: %+v", updated)
	}

	// ตรวจสอบความถูกต้องของอีเมลเมื่ออัปเดต
	_, err = svc.UpdateUser(ctx, created.ID.Hex(), service.UpdateUserRequest{
		Name:  "New Name",
		Email: "bad-email",
	})
	if err == nil {
		t.Error("expected error for invalid email, got nil")
	}
}

// TestDeleteUser ทดสอบการลบผู้ใช้
func TestDeleteUser(t *testing.T) {
	_, svc := setupTestEnv()
	ctx := context.Background()

	created, err := svc.Register(ctx, service.RegisterRequest{
		Name:     "To Delete",
		Email:    "delete@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// ลบสำเร็จ
	err = svc.DeleteUser(ctx, created.ID.Hex())
	if err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}

	// ตรวจสอบว่าถูกลบไปแล้วจริง
	_, err = svc.GetUserByID(ctx, created.ID.Hex())
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound after deletion, got %v", err)
	}

	// ลบซ้ำอีกครั้ง ต้องได้ ErrUserNotFound
	err = svc.DeleteUser(ctx, created.ID.Hex())
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound on deleting non-existent user, got %v", err)
	}
}

// TestCountUsers ทดสอบการนับจำนวนผู้ใช้สำหรับ Goroutine
func TestCountUsers(t *testing.T) {
	_, svc := setupTestEnv()
	ctx := context.Background()

	count, err := svc.CountUsers(ctx)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	svc.Register(ctx, service.RegisterRequest{Name: "User 1", Email: "u1@example.com", Password: "password123"})
	count, _ = svc.CountUsers(ctx)
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}
