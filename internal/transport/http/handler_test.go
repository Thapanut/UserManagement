package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-challenge/internal/pkg/jwt"
	"backend-challenge/internal/repository/mock"
	"backend-challenge/internal/service"
	httpTransport "backend-challenge/internal/transport/http"

	"github.com/go-chi/chi/v5"
)

func setupTestRouter() (http.Handler, *jwt.TokenManager) {
	repo := mock.NewMockUserRepository()
	tokenManager := jwt.NewTokenManager("test-jwt-secret-key-12345", 1)
	svc := service.NewUserService(repo, tokenManager)
	handler := httpTransport.NewUserHandler(svc, tokenManager)

	r := chi.NewRouter()
	r.Use(httpTransport.LoggingMiddleware)
	handler.RegisterRoutes(r)

	return r, tokenManager
}

// TestHealthCheckRoute ทดสอบ Endpoint /health
func TestHealthCheckRoute(t *testing.T) {
	router, _ := setupTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

// TestAuthRoutes_RegisterAndLogin ทดสอบ Register และ Login ผ่าน HTTP Handler
func TestAuthRoutes_RegisterAndLogin(t *testing.T) {
	router, _ := setupTestRouter()

	// 1. ทดสอบ Register
	registerPayload := map[string]string{
		"name":     "John Doe",
		"email":    "john@example.com",
		"password": "password123",
	}
	body, _ := json.Marshal(registerPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	// 2. ทดสอบ Login
	loginPayload := map[string]string{
		"email":    "john@example.com",
		"password": "password123",
	}
	body, _ = json.Marshal(loginPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var res struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Data.Token == "" {
		t.Errorf("expected JWT token string in login response")
	}

	// 3. ทดสอบเรียก Protected Route ด้วย Token
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+res.Data.Token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for protected route with token, got %d", rr.Code)
	}

	// 4. ทดสอบเรียก Protected Route โดยไม่มี Token (ต้องได้ 401 Unauthorized)
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized without token, got %d", rr.Code)
	}
}
