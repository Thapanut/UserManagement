package http

import (
	"encoding/json"
	"errors"
	"net/http"

	_ "backend-challenge/docs"
	"backend-challenge/internal/domain"
	"backend-challenge/internal/pkg/jwt"
	"backend-challenge/internal/service"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"
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

	// Swagger Documentation Routes
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", httpSwagger.WrapHandler)

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

// HealthCheck godoc
// @Summary ตรวจสอบสถานะความพร้อมของระบบ (Health Check)
// @Description ส่งกลับสถานะการทำงานของเซิร์ฟเวอร์
// @Tags System
// @Produce json
// @Success 200 {object} StandardResponse{data=map[string]string} "Server is healthy"
// @Router /health [get]
func (h *UserHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	SuccessResponse(w, http.StatusOK, map[string]string{"status": "UP"}, "server is healthy")
}

// Register godoc
// @Summary สมัครสมาชิกผู้ใช้ใหม่ (User Registration)
// @Description สร้างบัญชีผู้ใช้งานใหม่พร้อมแฮชรหัสผ่านด้วย bcrypt
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body service.RegisterRequest true "ข้อมูลการลงทะเบียน"
// @Success 201 {object} StandardResponse{data=domain.User} "User registered successfully"
// @Failure 400 {object} StandardResponse "Invalid request body or validation error"
// @Failure 409 {object} StandardResponse "Email already exists"
// @Router /auth/register [post]
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

// Login godoc
// @Summary เข้าสู่ระบบเพื่อรับ JWT Token (User Login)
// @Description ตรวจสอบอีเมลและรหัสผ่านเพื่อสร้าง JWT Token (HS256)
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "ข้อมูลเข้าสู่ระบบ"
// @Success 200 {object} StandardResponse{data=service.LoginResponse} "Login successful"
// @Failure 400 {object} StandardResponse "Invalid request body"
// @Failure 401 {object} StandardResponse "Invalid credentials"
// @Failure 500 {object} StandardResponse "Internal server error"
// @Router /auth/login [post]
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

// CreateUser godoc
// @Summary สร้างผู้ใช้ใหม่โดยตรง (Create User)
// @Description สร้างผู้ใช้ใหม่ในระบบ (Protected Endpoint)
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.CreateUserRequest true "ข้อมูลผู้ใช้ใหม่"
// @Success 201 {object} StandardResponse{data=domain.User} "User created successfully"
// @Failure 400 {object} StandardResponse "Bad request or validation error"
// @Failure 401 {object} StandardResponse "Unauthorized"
// @Failure 409 {object} StandardResponse "Email already exists"
// @Router /users [post]
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

// ListUsers godoc
// @Summary ดึงรายชื่อผู้ใช้ทั้งหมด (List Users)
// @Description ดึงรายการผู้ใช้งานทั้งหมดในระบบ (Protected Endpoint)
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} StandardResponse{data=[]domain.User} "Users retrieved successfully"
// @Failure 401 {object} StandardResponse "Unauthorized"
// @Failure 500 {object} StandardResponse "Internal server error"
// @Router /users [get]
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userService.ListUsers(r.Context())
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "failed to retrieve users")
		return
	}

	SuccessResponse(w, http.StatusOK, users, "users retrieved successfully")
}

// GetUserByID godoc
// @Summary ดึงข้อมูลผู้ใช้ตาม ID (Get User by ID)
// @Description ค้นหาข้อมูลผู้ใช้จาก MongoDB ObjectID Hex string (Protected Endpoint)
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param id path string true "User ObjectID Hex String" example("6aacb46c71edc7877f277dae")
// @Success 200 {object} StandardResponse{data=domain.User} "User retrieved successfully"
// @Failure 400 {object} StandardResponse "Invalid ID format"
// @Failure 401 {object} StandardResponse "Unauthorized"
// @Failure 404 {object} StandardResponse "User not found"
// @Failure 500 {object} StandardResponse "Internal server error"
// @Router /users/{id} [get]
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

// UpdateUser godoc
// @Summary อัปเดตข้อมูลผู้ใช้ (Update User)
// @Description อัปเดตชื่อ และ/หรือ อีเมลของผู้ใช้ รองรับ Partial Update (Protected Endpoint)
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User ObjectID Hex String" example("6aacb46c71edc7877f277dae")
// @Param request body service.UpdateUserRequest true "ฟิลด์ที่ต้องการอัปเดต"
// @Success 200 {object} StandardResponse{data=domain.User} "User updated successfully"
// @Failure 400 {object} StandardResponse "Bad request or validation error"
// @Failure 401 {object} StandardResponse "Unauthorized"
// @Failure 404 {object} StandardResponse "User not found"
// @Failure 409 {object} StandardResponse "Email already exists"
// @Router /users/{id} [put]
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

// DeleteUser godoc
// @Summary ลบผู้ใช้ (Delete User)
// @Description ลบผู้ใช้ออกจากระบบตาม ID (Protected Endpoint)
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param id path string true "User ObjectID Hex String" example("6aacb46c71edc7877f277dae")
// @Success 200 {object} StandardResponse "User deleted successfully"
// @Failure 400 {object} StandardResponse "Invalid ID format"
// @Failure 401 {object} StandardResponse "Unauthorized"
// @Failure 404 {object} StandardResponse "User not found"
// @Failure 500 {object} StandardResponse "Internal server error"
// @Router /users/{id} [delete]
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
