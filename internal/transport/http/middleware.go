package http

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-challenge/internal/pkg/jwt"
)

// contextKey ชนิดข้อมูลเฉพาะสำหรับ key ใน context เพื่อป้องกันการชนกันของ key
type contextKey string

const (
	// UserIDContextKey คีย์สำหรับเก็บ UserID ใน context หลังจากผ่าน JWT Auth
	UserIDContextKey contextKey = "userID"

	// UserEmailContextKey คีย์สำหรับเก็บ Email ใน context
	UserEmailContextKey contextKey = "userEmail"
)

// responseWriterWrapper ครอบ http.ResponseWriter เพื่อดักจับ HTTP Status Code ที่ถูกตอบกลับ
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware ดักจับและบันทึกข้อมูลการเรียกใช้งาน API:
// - HTTP Method (เช่น GET, POST)
// - Request Path (เช่น /api/v1/users)
// - Execution Time (ระยะเวลาที่ใช้ในการประมวลผลคำขอ)
// - Response Status Code
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // ค่าเริ่มต้นถ้า handler ไม่ได้เรียก WriteHeader
		}

		// ส่งคำขอต่อไปยัง handler ถัดไป
		next.ServeHTTP(wrapper, r)

		// คำนวณระยะเวลาการประมวลผล
		duration := time.Since(start)

		// แสดง log ในรูปแบบที่อ่านง่ายและชัดเจน
		log.Printf("[HTTP] %s %s | Status: %d | Duration: %v | ClientIP: %s",
			r.Method,
			r.URL.Path,
			wrapper.statusCode,
			duration,
			r.RemoteAddr,
		)
	})
}

// JWTMiddleware ตรวจสอบความถูกต้องของ JWT Token ใน Authorization Header
// ป้องกัน Endpoint ที่ต้องการ Authentication
func JWTMiddleware(tokenManager *jwt.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader == "" {
				ErrorResponse(w, http.StatusUnauthorized, "authorization header required")
				return
			}

			// รองรับทั้ง "Bearer <token>" (RFC 6750) และการใส่ raw token โดยตรง (เช่น จาก Swagger UI หรือ API Client)
			tokenString := authHeader
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				tokenString = strings.TrimSpace(authHeader[7:])
			}

			if tokenString == "" {
				ErrorResponse(w, http.StatusUnauthorized, "authorization token is empty")
				return
			}

			claims, err := tokenManager.ValidateToken(tokenString)
			if err != nil {
				ErrorResponse(w, http.StatusUnauthorized, err.Error())
				return
			}

			// นำ UserID และ Email ไปเก็บไว้ใน request context
			// เพื่อให้ handler ตัวถัดไปสามารถดึงข้อมูลผู้ใช้งานที่ login อยู่มาใช้งานได้
			ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailContextKey, claims.Email)

			// เรียก handler ตัวถัดไปพร้อม context ใหม่
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext ดึง UserID จาก Context
func GetUserIDFromContext(ctx context.Context) string {
	if val, ok := ctx.Value(UserIDContextKey).(string); ok {
		return val
	}
	return ""
}

// CORSMiddleware จัดการ Cross-Origin Resource Sharing (CORS) สำหรับ Frontend Web Apps, Postman, และ Swagger UI
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Expose-Headers", "Link")
		w.Header().Set("Access-Control-Max-Age", "300")

		// ตอบกลับทันทีหากเป็น Preflight Request (OPTIONS)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

