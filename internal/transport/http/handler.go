package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"backend-challenge/internal/domain"
	"backend-challenge/internal/pkg/jwt"
	"backend-challenge/internal/service"

	"github.com/go-chi/chi/v5"
)

// UserHandler โครงสร้างจัดการ HTTP Requests สำหรับ User และ Authentication
type UserHandler struct {
	userService  *service.UserService
	tokenManager *jwt.TokenManager
}

// NewUserHandler สร้าง Instance ใหม่ของ UserHandler
func NewUserHandler(userService *service.UserService, tokenManager *jwt.TokenManager) *UserHandler {
	return &UserHandler{
		userService:  userService,
		tokenManager: tokenManager,
	}
}

// RegisterRoutes ลงทะเบียนเส้นทาง (Routes) ทั้งหมดของระบบ
func (h *UserHandler) RegisterRoutes(r chi.Router) {
	// Root and Health check
	r.Get("/health", h.HealthCheck)

	// API Version 1
	r.Route("/api/v1", func(r chi.Router) {
		// เส้นทางสาธารณะสำหรับการยืนยันตัวตน (Public Auth Routes)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", h.Register)
			r.Post("/login", h.Login)
		})

		// เส้นทางการจัดการข้อมูลผู้ใช้ (Protected User Routes)
		// ทุก Endpoint ในกลุ่มนี้ต้องผ่าน JWTMiddleware ก่อน
		r.Group(func(r chi.Router) {
			r.Use(JWTMiddleware(h.tokenManager))

			r.Route("/users", func(r chi.Router) {
				r.Post("/", h.CreateUser)       // สร้างผู้ใช้ใหม่
				r.Get("/", h.ListUsers)         // ดูรายชื่อผู้ใช้ทั้งหมด
				r.Get("/{id}", h.GetUserByID)   // ดูข้อมูลผู้ใช้ตาม ID
				r.Put("/{id}", h.UpdateUser)    // อัปเดตชื่อหรืออีเมล
				r.Delete("/{id}", h.DeleteUser) // ลบผู้ใช้
			})
		})
	})
}

// HealthCheck ตรวจสอบสถานะความพร้อมของ API Server
func (h *UserHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	SuccessResponse(w, http.StatusOK, map[string]string{"status": "UP"}, "server is healthy")
}

// Register จัดการการลงทะเบียนผู้ใช้งานใหม่ (POST /api/v1/auth/register)
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	user, err := h.userService.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			ErrorResponse(w, http.StatusConflict, err.Error())
			return
		}
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	SuccessResponse(w, http.StatusCreated, user, "user registered successfully")
}

// Login ตรวจสอบความถูกต้องของบัญชีและออก JWT Token (POST /api/v1/auth/login)
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req service.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	loginRes, err := h.userService.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			ErrorResponse(w, http.StatusUnauthorized, err.Error())
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, "failed to authenticate")
		return
	}

	SuccessResponse(w, http.StatusOK, loginRes, "login successful")
}

// CreateUser สร้างผู้ใช้ใหม่โดยตรง (POST /api/v1/users) [Protected]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req service.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	user, err := h.userService.CreateUser(r.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			ErrorResponse(w, http.StatusConflict, err.Error())
			return
		}
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	SuccessResponse(w, http.StatusCreated, user, "user created successfully")
}

// ListUsers ดึงรายชื่อผู้ใช้ทั้งหมด (GET /api/v1/users) [Protected]
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userService.ListUsers(r.Context())
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to retrieve users")
		return
	}

	SuccessResponse(w, http.StatusOK, users, "users retrieved successfully")
}

// GetUserByID ดึงข้อมูลผู้ใช้ตาม ID (GET /api/v1/users/{id}) [Protected]
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := h.userService.GetUserByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidID) {
			ErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			ErrorResponse(w, http.StatusNotFound, err.Error())
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, "failed to retrieve user")
		return
	}

	SuccessResponse(w, http.StatusOK, user, "user retrieved successfully")
}

// UpdateUser แก้ไขชื่อหรืออีเมลของผู้ใช้ (PUT /api/v1/users/{id}) [Protected]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req service.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "invalid request body format")
		return
	}

	updatedUser, err := h.userService.UpdateUser(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidID) {
			ErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			ErrorResponse(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			ErrorResponse(w, http.StatusConflict, err.Error())
			return
		}
		ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	SuccessResponse(w, http.StatusOK, updatedUser, "user updated successfully")
}

// DeleteUser ลบผู้ใช้ออกจากระบบ (DELETE /api/v1/users/{id}) [Protected]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.userService.DeleteUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidID) {
			ErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			ErrorResponse(w, http.StatusNotFound, err.Error())
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	SuccessResponse(w, http.StatusOK, nil, "user deleted successfully")
}
