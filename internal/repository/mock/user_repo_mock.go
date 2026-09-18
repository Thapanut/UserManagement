package mock

import (
	"context"
	"sync"
	"time"

	"backend-challenge/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockUserRepository คือ In-Memory Mock ของ domain.UserRepository
// ออกแบบมาเพื่อใช้ในการทำ Unit Tests โดยไม่ต้องเชื่อมต่อกับ MongoDB จริง
type MockUserRepository struct {
	mu    sync.RWMutex
	users map[primitive.ObjectID]*domain.User

	// Hook functions สำหรับจำลองข้อผิดพลาดในการทดสอบ (Error Injection)
	CreateFunc      func(ctx context.Context, user *domain.User) error
	FindByIDFunc    func(ctx context.Context, id primitive.ObjectID) (*domain.User, error)
	FindByEmailFunc func(ctx context.Context, email string) (*domain.User, error)
	FindAllFunc     func(ctx context.Context) ([]*domain.User, error)
	UpdateFunc      func(ctx context.Context, id primitive.ObjectID, name, email string) (*domain.User, error)
	DeleteFunc      func(ctx context.Context, id primitive.ObjectID) error
	CountFunc       func(ctx context.Context) (int64, error)
}

// NewMockUserRepository สร้าง Instance ใหม่ของ MockUserRepository
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[primitive.ObjectID]*domain.User),
	}
}

// Create จำลองการบันทึกผู้ใช้ลง map
func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, user)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// ตรวจสอบว่า email ซ้ำหรือไม่
	for _, u := range m.users {
		if u.Email == user.Email {
			return domain.ErrEmailAlreadyExists
		}
	}

	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}

	// คัดลอกข้อมูลเพื่อป้องกัน race condition
	userCopy := *user
	m.users[user.ID] = &userCopy
	return nil
}

// FindByID จำลองการค้นหาตาม ID
func (m *MockUserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.User, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	u, exists := m.users[id]
	if !exists {
		return nil, domain.ErrUserNotFound
	}

	userCopy := *u
	return &userCopy, nil
}

// FindByEmail จำลองการค้นหาตาม Email
func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.FindByEmailFunc != nil {
		return m.FindByEmailFunc(ctx, email)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, u := range m.users {
		if u.Email == email {
			userCopy := *u
			return &userCopy, nil
		}
	}

	return nil, domain.ErrUserNotFound
}

// FindAll จำลองการดึงผู้ใช้ทั้งหมด
func (m *MockUserRepository) FindAll(ctx context.Context) ([]*domain.User, error) {
	if m.FindAllFunc != nil {
		return m.FindAllFunc(ctx)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*domain.User, 0, len(m.users))
	for _, u := range m.users {
		userCopy := *u
		result = append(result, &userCopy)
	}

	return result, nil
}

// Update จำลองการอัปเดตข้อมูลผู้ใช้
func (m *MockUserRepository) Update(ctx context.Context, id primitive.ObjectID, name, email string) (*domain.User, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, name, email)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	u, exists := m.users[id]
	if !exists {
		return nil, domain.ErrUserNotFound
	}

	// ตรวจสอบว่าอีเมลใหม่ชนกับ user คนอื่นหรือไม่
	for otherID, otherUser := range m.users {
		if otherID != id && otherUser.Email == email {
			return nil, domain.ErrEmailAlreadyExists
		}
	}

	u.Name = name
	u.Email = email

	userCopy := *u
	return &userCopy, nil
}

// Delete จำลองการลบผู้ใช้
func (m *MockUserRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[id]; !exists {
		return domain.ErrUserNotFound
	}

	delete(m.users, id)
	return nil
}

// Count จำลองการนับจำนวนผู้ใช้ทั้งหมด
func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	if m.CountFunc != nil {
		return m.CountFunc(ctx)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return int64(len(m.users)), nil
}
