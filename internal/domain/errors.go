package domain

import "errors"

// รายการ Domain Errors สำหรับจัดการข้อผิดพลาดต่างๆ อย่างเป็นระบบ
var (
	// ErrUserNotFound เมื่อไม่พบผู้ใช้ในระบบ
	ErrUserNotFound = errors.New("user not found")

	// ErrEmailAlreadyExists เมื่อมีอีเมลนี้อยู่ในระบบแล้ว
	ErrEmailAlreadyExists = errors.New("email already exists")

	// ErrInvalidCredentials เมื่ออีเมลหรือรหัสผ่านไม่ถูกต้อง
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrInvalidID เมื่อ ID ที่ส่งเข้ามาไม่ถูกต้องตามรูปแบบ ObjectID ของ MongoDB
	ErrInvalidID = errors.New("invalid user ID format")

	// ErrValidationFailed เมื่อข้อมูลที่ส่งเข้ามาไม่ผ่านการตรวจสอบ (เช่น กรอกไม่ครบ หรืออีเมลผิดฟอร์แมต)
	ErrValidationFailed = errors.New("validation failed")
)
