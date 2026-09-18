package domain

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User โครงสร้างข้อมูลผู้ใช้งาน (Entity)
// ตามข้อกำหนดใน README:
// - ID (auto-generated โดย MongoDB ObjectID)
// - Name (string)
// - Email (string, unique)
// - Password (hashed ด้วย bcrypt ไม่ส่งออกไปยัง JSON response)
// - CreatedAt (timestamp วันที่สร้าง)
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id" swaggertype:"string" example:"6aacb46c71edc7877f277dae"`
	Name      string             `bson:"name" json:"name" example:"Somchai Jaidee"`
	Email     string             `bson:"email" json:"email" example:"somchai@example.com"`
	Password  string             `bson:"password" json:"-"` // เครื่องหมาย "-" ซ่อนรหัสผ่านไม่ให้แสดงใน JSON response
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// UserRepository Interface กำหนดฟังก์ชันสำหรับการติดต่อ Database
// การใช้ Interface ช่วยเรื่อง Abstraction และทำให้ทำ Mock Unit Test ได้ง่าย (Bonus Feature)
type UserRepository interface {
	// Create บันทึกผู้ใช้ใหม่ลงฐานข้อมูล
	Create(ctx context.Context, user *User) error

	// FindByID ค้นหาผู้ใช้จาก ID
	FindByID(ctx context.Context, id primitive.ObjectID) (*User, error)

	// FindByEmail ค้นหาผู้ใช้จาก Email (สำหรับ Login และตรวจสอบความซ้ำซ้อน)
	FindByEmail(ctx context.Context, email string) (*User, error)

	// FindAll ดึงรายชื่อผู้ใช้ทั้งหมด
	FindAll(ctx context.Context) ([]*User, error)

	// Update อัปเดตข้อมูล Name และ Email ของผู้ใช้
	Update(ctx context.Context, id primitive.ObjectID, name string, email string) (*User, error)

	// Delete ลบผู้ใช้ออกจากฐานข้อมูลตาม ID
	Delete(ctx context.Context, id primitive.ObjectID) error

	// Count คืนค่าจำนวนผู้ใช้ทั้งหมดในฐานข้อมูล (สำหรับ Background Goroutine)
	Count(ctx context.Context) (int64, error)
}
