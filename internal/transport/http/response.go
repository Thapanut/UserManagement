package http

import (
	"encoding/json"
	"net/http"
)

// StandardResponse รูปแบบ JSON Response มาตรฐานของ API
type StandardResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// JSON ตอบกลับ Client ด้วย Content-Type เป็น application/json
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// SuccessResponse คืนผลลัพธ์สำเร็จ
func SuccessResponse(w http.ResponseWriter, statusCode int, data interface{}, message string) {
	JSON(w, statusCode, StandardResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// ErrorResponse คืนผลลัพธ์กรณีเกิดข้อผิดพลาด
func ErrorResponse(w http.ResponseWriter, statusCode int, errMsg string) {
	JSON(w, statusCode, StandardResponse{
		Success: false,
		Error:   errMsg,
	})
}
