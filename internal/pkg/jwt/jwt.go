package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken เมื่อ Token ไม่ถูกต้อง หรือ signature ไม่ตรง
	ErrInvalidToken = errors.New("invalid or malformed token")

	// ErrExpiredToken เมื่อ Token หมดอายุแล้ว
	ErrExpiredToken = errors.New("token has expired")
)

// CustomClaims โครงสร้าง Claims ที่บันทึกข้อมูลผู้ใช้ลงใน JWT Payload
type CustomClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// TokenManager จัดการการสร้างและตรวจสอบ JWT Token
type TokenManager struct {
	secretKey []byte
	duration  time.Duration
}

// NewTokenManager สร้าง Instance ใหม่ของ TokenManager
func NewTokenManager(secretKey string, durationHours int) *TokenManager {
	return &TokenManager{
		secretKey: []byte(secretKey),
		duration:  time.Duration(durationHours) * time.Hour,
	}
}

// GenerateToken สร้าง JWT Token ที่ลงนามด้วย HMAC-SHA256 (HS256)
func (m *TokenManager) GenerateToken(userID string, email string) (string, error) {
	now := time.Now()
	claims := CustomClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.duration)),
			Issuer:    "backend-challenge-api",
		},
	}

	// สร้าง Token ด้วยวิธี HS256 (HMAC with SHA-256) ตามที่กำหนดใน README
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// เซ็น Token ด้วย Secret Key
	signedToken, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// ValidateToken ตรวจสอบความถูกต้องและแกะ Claims ออกจาก JWT Token
func (m *TokenManager) ValidateToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// ตรวจสอบว่า Signing Method เป็น HMAC ตามที่คาดหวังหรือไม่
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
