package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"backend-challenge/internal/domain"
	"backend-challenge/internal/pkg/hasher"
	"backend-challenge/internal/pkg/jwt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterRequest ข้อมูลสำหรับการลงทะเบียนผู้ใช้ใหม่
type RegisterRequest struct {
	Name     string `json:"name" example:"Somchai Jaidee"`
	Email    string `json:"email" example:"somchai@example.com"`
	Password string `json:"password" example:"password123"`
}

// LoginRequest ข้อมูลสำหรับการเข้าสู่ระบบ
type LoginRequest struct {
	Email    string `json:"email" example:"somchai@example.com"`
	Password string `json:"password" example:"password123"`
}

// LoginResponse ผลลัพธ์จากการเข้าสู่ระบบสำเร็จ คืนค่า JWT Token
type LoginResponse struct {
	Token string       `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User  *domain.User `json:"user"`
}

// CreateUserRequest ข้อมูลสำหรับการสร้างผู้ใช้
type CreateUserRequest struct {
	Name     string `json:"name" example:"Somchai Jaidee"`
	Email    string `json:"email" example:"somchai@example.com"`
	Password string `json:"password" example:"password123"`
}

// UpdateUserRequest ข้อมูลสำหรับการอัปเดตชื่อหรืออีเมล
type UpdateUserRequest struct {
	Name  string `json:"name" example:"Somchai Pro"`
	Email string `json:"email" example:"somchai.pro@example.com"`
}

// UserService โครงสร้าง Business Logic Service
type UserService struct {
	repo         domain.UserRepository
	tokenManager *jwt.TokenManager
}

// NewUserService สร้าง Instance ใหม่ของ UserService
func NewUserService(repo domain.UserRepository, tokenManager *jwt.TokenManager) *UserService {
	return &UserService{
		repo:         repo,
		tokenManager: tokenManager,
	}
}

// validateEmail ตรวจสอบความถูกต้องของรูปแบบ Email (RFC 5322)
func validateEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	_, err := mail.ParseAddress(email)
	return err == nil
}

// Register ลงทะเบียนผู้ใช้ใหม่:
// 1. ตรวจสอบความถูกต้องของ Input (Validation)
// 2. ตรวจสอบว่ามี Email นี้ในระบบแล้วหรือไม่
// 3. แฮชรหัสผ่านด้วย bcrypt
// 4. บันทึกลงฐานข้อมูล
func (s *UserService) Register(ctx context.Context, req RegisterRequest) (*domain.User, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password

	// ตรวจสอบความถูกต้องของข้อมูล
	if name == "" {
		return nil, errors.New("name is required")
	}
	if !validateEmail(email) {
		return nil, errors.New("invalid email address format")
	}
	if len(password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}

	// ตรวจสอบความซ้ำซ้อนของอีเมล
	existing, err := s.repo.FindByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, domain.ErrEmailAlreadyExists
	}
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, err
	}

	// แฮชรหัสผ่านด้วย bcrypt
	hashedPassword, err := hasher.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Name:     name,
		Email:    email,
		Password: hashedPassword,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login เข้าสู่ระบบ:
// 1. ค้นหาผู้ใช้จาก Email
// 2. ตรวจสอบรหัสผ่าน bcrypt
// 3. สร้างและคืนค่า JWT Token (HMAC-SHA256)
func (s *UserService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := req.Password

	if email == "" || password == "" {
		return nil, domain.ErrInvalidCredentials
	}

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	// เปรียบเทียบรหัสผ่าน
	if !hasher.CheckPassword(password, user.Password) {
		return nil, domain.ErrInvalidCredentials
	}

	// สร้าง JWT Token
	token, err := s.tokenManager.GenerateToken(user.ID.Hex(), user.Email)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

// CreateUser สร้างผู้ใช้ใหม่ (สำหรับการจัดการ User โดยตรง)
func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*domain.User, error) {
	return s.Register(ctx, RegisterRequest{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
}

// GetUserByID ค้นหาผู้ใช้ตาม ObjectID Hex String
func (s *UserService) GetUserByID(ctx context.Context, idHex string) (*domain.User, error) {
	objID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return nil, domain.ErrInvalidID
	}

	return s.repo.FindByID(ctx, objID)
}

// ListUsers คืนรายการผู้ใช้ทั้งหมด
func (s *UserService) ListUsers(ctx context.Context) ([]*domain.User, error) {
	return s.repo.FindAll(ctx)
}

// UpdateUser อัปเดตข้อมูล Name และ/หรือ Email ของผู้ใช้ (รองรับ Partial Update ตามโจทย์ "name or email")
func (s *UserService) UpdateUser(ctx context.Context, idHex string, req UpdateUserRequest) (*domain.User, error) {
	objID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return nil, domain.ErrInvalidID
	}

	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// ตรวจสอบว่ามีการส่งอย่างน้อย 1 ฟิลด์หรือไม่
	if name == "" && email == "" {
		return nil, errors.New("at least one field (name or email) must be provided for update")
	}

	// ตรวจสอบว่าผู้ใช้มีอยู่จริงก่อนทำการอัปเดต
	existingUser, err := s.repo.FindByID(ctx, objID)
	if err != nil {
		return nil, err
	}

	// หากไม่ได้ส่ง Name มา ให้ใช้ Name เดิมของผู้ใช้
	if name == "" {
		name = existingUser.Name
	}

	// หากไม่ได้ส่ง Email มา ให้ใช้ Email เดิมของผู้ใช้ แต่หากส่งมาให้ตรวจสอบ Format
	if email == "" {
		email = existingUser.Email
	} else {
		if !validateEmail(email) {
			return nil, errors.New("invalid email address format")
		}
	}

	return s.repo.Update(ctx, objID, name, email)
}

// DeleteUser ลบผู้ใช้ตาม ID
func (s *UserService) DeleteUser(ctx context.Context, idHex string) error {
	objID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return domain.ErrInvalidID
	}

	return s.repo.Delete(ctx, objID)
}

// CountUsers นับจำนวนผู้ใช้ทั้งหมด (สำหรับ Goroutine)
func (s *UserService) CountUsers(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}
