package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend-challenge/internal/config"
	"backend-challenge/internal/pkg/jwt"
	"backend-challenge/internal/repository/mongodb"
	"backend-challenge/internal/service"
	grpcTransport "backend-challenge/internal/transport/grpc"
	httpTransport "backend-challenge/internal/transport/http"
	"backend-challenge/internal/worker"
	pb "backend-challenge/proto"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// @title User Management API
// @version 1.0
// @description Backend Challenge User Management RESTful API with Hexagonal Architecture & JWT Authentication.
// @contact.name Thapanut
// @license.name Apache 2.0
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your JWT token directly or with Bearer prefix (e.g. 'Bearer <token>' or just '<token>').

func main() {
	// 1. โหลดการตั้งค่าจาก Environment Variables
	cfg := config.LoadConfig()
	log.Printf("==================================================")
	log.Printf("Starting User Management Service (REST & gRPC)")
	log.Printf("HTTP Port: %s | gRPC Port: %s", cfg.Port, cfg.GRPCPort)
	log.Printf("Mongo URI: %s | DB Name: %s", cfg.MongoURI, cfg.DBName)
	log.Printf("==================================================")

	// 2. เชื่อมต่อกับ MongoDB ผ่าน Official Go MongoDB Driver
	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mongoCancel()

	clientOptions := options.Client().ApplyURI(cfg.MongoURI)
	mongoClient, err := mongo.Connect(mongoCtx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// ตรวจสอบการเชื่อมต่อ (Ping)
	if err := mongoClient.Ping(mongoCtx, readpref.Primary()); err != nil {
		log.Fatalf("MongoDB ping failed: %v", err)
	}
	log.Println("Successfully connected to MongoDB")

	db := mongoClient.Database(cfg.DBName)

	// 3. เริ่มต้น Layer ต่างๆ ตามหลัก Hexagonal Architecture (Ports & Adapters)
	// 3.1 Repository Layer (Adapter ต่อฐานข้อมูล)
	userRepo, err := mongodb.NewMongoUserRepository(db)
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB user repository: %v", err)
	}

	// 3.2 Security / Token Manager
	tokenManager := jwt.NewTokenManager(cfg.JWTSecret, cfg.JWTExpirationHours)

	// 3.3 Service Layer (Core Business Logic)
	userService := service.NewUserService(userRepo, tokenManager)

	// 3.4 Transport Layer (HTTP Handler)
	userHandler := httpTransport.NewUserHandler(userService, tokenManager)

	// 4. ตั้งค่า HTTP Router (chi) และ Middleware
	r := chi.NewRouter()

	// CORS Middleware รองรับการเรียกจาก Browser และ Swagger UI
	r.Use(httpTransport.CORSMiddleware)

	// Logging Middleware ตาม Requirement 5 (บันทึก method, path, execution time)
	r.Use(httpTransport.LoggingMiddleware)

	// Middleware ป้องกันเซิร์ฟเวอร์แครชจาก Panic
	r.Use(chimiddleware.Recoverer)

	// ลงทะเบียน Routes ทั้งหมด
	userHandler.RegisterRoutes(r)

	// 5. Concurrency Task: Background Goroutine ตาม Requirement 6
	// รันทุกๆ 10 วินาทีเพื่อนับจำนวนผู้ใช้ในฐานข้อมูล
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()

	counterWorker := worker.NewUserCounterWorker(userService, cfg.UserCountIntervalSeconds)
	go counterWorker.Start(workerCtx)

	// 6. สตาร์ท HTTP Server (Port 8080)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("HTTP REST API Server listening on port %s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 7. สตาร์ท gRPC Server (Port 50051) [Bonus Feature]
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcTransport.AuthInterceptor(tokenManager)),
	)
	pb.RegisterUserServiceServer(grpcServer, grpcTransport.NewUserGRPCServer(userService))

	// เปิด gRPC Server Reflection เพื่อให้เครื่องมืออย่าง grpcurl หรือ Postman ตรวจสอบ schema ได้อัตโนมัติ
	reflection.Register(grpcServer)

	grpcLis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port %s: %v", cfg.GRPCPort, err)
	}

	go func() {
		log.Printf("gRPC Server listening on port %s", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// 8. Graceful Shutdown (Bonus Feature)
	// ดักจับสัญญาณ SIGINT (Ctrl+C) หรือ SIGTERM จากระบบปฏิบัติการหรือ Docker
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// รอรับสัญญาณ
	sig := <-quit
	log.Printf("Received shutdown signal: %v. Initiating graceful shutdown...", sig)

	// 8.1 หยุดการทำงานของ Background Goroutine
	cancelWorker()

	// 8.2 ปิด gRPC Server
	log.Println("Stopping gRPC server...")
	grpcServer.GracefulStop()
	log.Println("gRPC server stopped successfully")

	// 8.3 ปิด HTTP Server แบบ Graceful ภายในเวลา 10 วินาที
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	} else {
		log.Println("HTTP server shut down gracefully")
	}

	// 8.4 ปิดการเชื่อมต่อ MongoDB Client
	disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer disconnectCancel()

	if err := mongoClient.Disconnect(disconnectCtx); err != nil {
		log.Printf("Error disconnecting from MongoDB: %v", err)
	} else {
		log.Println("Disconnected from MongoDB")
	}

	log.Println("Server exited cleanly")
}
