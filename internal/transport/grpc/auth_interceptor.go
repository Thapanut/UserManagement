package grpc

import (
	"context"
	"strings"

	"backend-challenge/internal/pkg/jwt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor ทำหน้าที่เป็น gRPC Unary Interceptor
// สำหรับตรวจสอบ JWT token จาก gRPC metadata (Authorization: Bearer <token>)
func AuthInterceptor(tokenManager *jwt.TokenManager) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// ข้ามการตรวจ Token สำหรับ CreateUser เพื่อให้ผู้ใช้ใหม่สามารถสมัครสมาชิกผ่าน gRPC ได้
		if info.FullMethod == "/user.UserService/CreateUser" {
			return handler(ctx, req)
		}

		// ดึง Metadata จาก context สำหรับ Endpoint ที่ต้องการการยืนยันตัวตน (เช่น GetUser)
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
		}

		values := md["authorization"]
		if len(values) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
		}

		authHeader := strings.TrimSpace(values[0])
		tokenString := authHeader
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			tokenString = strings.TrimSpace(authHeader[7:])
		}

		if tokenString == "" {
			return nil, status.Errorf(codes.Unauthenticated, "authorization token is empty")
		}
		claims, err := tokenManager.ValidateToken(tokenString)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid or expired token: %v", err)
		}

		// ส่งผ่านไปยัง Handler พร้อม Context
		newCtx := context.WithValue(ctx, "userID", claims.UserID)
		return handler(newCtx, req)
	}
}
