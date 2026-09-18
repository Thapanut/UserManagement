package hasher

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword แฮชรหัสผ่าน plain text โดยใช้ bcrypt ด้วย default cost (10)
// ช่วยให้ปลอดภัยและป้องกัน rainbow table attacks ด้วย salt อัตโนมัติ
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword ตรวจสอบว่า plain password ตรงกับ hashed password หรือไม่
// คืนค่า true หากตรงกัน และ false หากไม่ตรง
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
