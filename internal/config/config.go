package config

import (
	"os"
	"strconv"
)

// Config เก็บการตั้งค่าทั้งหมดของแอปพลิเคชัน
// ค่าเริ่มต้นจะถูกกำหนดไว้ และสามารถ override ผ่าน Environment Variables ได้
type Config struct {
	// Port คือพอร์ตที่ HTTP server จะเปิดทำงาน
	Port string

	// GRPCPort คือพอร์ตที่ gRPC server จะเปิดทำงาน
	GRPCPort string

	// MongoURI คือ connection string สำหรับเชื่อมต่อ MongoDB
	MongoURI string

	// DBName คือชื่อฐานข้อมูล MongoDB
	DBName string

	// JWTSecret คือ Secret Key ที่ใช้สำหรับ Sign และ Verify HMAC-SHA256 tokens
	JWTSecret string

	// JWTExpirationHours คืออายุของ Token เป็นจำนวนชั่วโมง
	JWTExpirationHours int

	// UserCountIntervalSeconds คือความถี่ในการรัน Background Goroutine เพื่อนับจำนวนผู้ใช้
	UserCountIntervalSeconds int
}

// LoadConfig อ่านค่าจาก Environment Variables หากไม่มีจะใช้ค่า Default
func LoadConfig() *Config {
	return &Config{
		Port:                     getEnv("PORT", "8080"),
		GRPCPort:                 getEnv("GRPC_PORT", "50051"),
		MongoURI:                 getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:                   getEnv("DB_NAME", "user_management_db"),
		JWTSecret:                getEnv("JWT_SECRET", "super-secret-key-change-in-production-12345"),
		JWTExpirationHours:       getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		UserCountIntervalSeconds: getEnvAsInt("USER_COUNT_INTERVAL_SECONDS", 10),
	}
}

// getEnv ดึงค่า string จาก env ถ้าไม่พบจะคืน fallback value
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// getEnvAsInt ดึงค่า integer จาก env ถ้าไม่พบหรือแปลงไม่สำเร็จจะคืน fallback value
func getEnvAsInt(key string, fallback int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return fallback
	}
	return val
}
