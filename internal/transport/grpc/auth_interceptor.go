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
		// ดึง Metadata จาก context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
		}

		values := md["authorization"]
		if len(values) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
		}

		authHeader := values[0]
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization token format")
		}

		tokenString := strings.TrimSpace(parts[1])
		claims, err := tokenManager.ValidateToken(tokenString)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid or expired token: %v", err)
		}

		// ส่งผ่านไปยัง Handler พร้อม Context
		newCtx := context.WithValue(ctx, "userID", claims.UserID)
		return handler(newCtx, req)
	}
}
