package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend-challenge/internal/config"
	"backend-challenge/internal/pkg/jwt"
	"backend-challenge/internal/repository/mongodb"
	"backend-challenge/internal/service"
	grpcTransport "backend-challenge/internal/transport/grpc"
	pb "backend-challenge/proto"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("==================================================")
	log.Printf("🚀 Starting User Management gRPC Service")
	log.Printf("   gRPC Port: %s", cfg.GRPCPort)
	log.Printf("   Mongo URI: %s", cfg.MongoURI)
	log.Printf("==================================================")

	// เชื่อมต่อ MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.MongoURI)
	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("❌ Failed to connect to MongoDB: %v", err)
	}

	if err := mongoClient.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatalf("❌ MongoDB ping failed: %v", err)
	}

	db := mongoClient.Database(cfg.DBName)
	userRepo, err := mongodb.NewMongoUserRepository(db)
	if err != nil {
		log.Fatalf("❌ Failed to initialize repository: %v", err)
	}

	tokenManager := jwt.NewTokenManager(cfg.JWTSecret, cfg.JWTExpirationHours)
	userService := service.NewUserService(userRepo, tokenManager)
	grpcServerImpl := grpcTransport.NewUserGRPCServer(userService)

	// สร้าง gRPC Server พร้อม Auth Interceptor (Secured with token metadata)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcTransport.AuthInterceptor(tokenManager)),
	)
	pb.RegisterUserServiceServer(grpcServer, grpcServerImpl)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("❌ Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	go func() {
		log.Printf("📡 gRPC Server listening on port %s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("❌ gRPC server error: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("⚠️ Shutting down gRPC server gracefully...")
	grpcServer.GracefulStop()

	_ = mongoClient.Disconnect(context.Background())
	log.Println("✅ gRPC server stopped successfully")
}
