# ==========================================
# Stage 1: Build stage
# ==========================================
FROM golang:1.24-alpine AS builder

WORKDIR /app

# คัดลอก go.mod และ go.sum เพื่อทำการ cache layer dependencies
COPY go.mod go.sum ./
RUN go mod download

# คัดลอกซอร์สโค้ดทั้งหมด
COPY . .

# คอมไพล์ Go binary ให้เป็นแบบ Statically Linked (CGO_ENABLED=0)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api

# ==========================================
# Stage 2: Minimal Runtime stage
# ==========================================
FROM alpine:3.20

WORKDIR /app

# ติดตั้ง tzdata และ ca-certificates สำหรับการเชื่อมต่อ
RUN apk --no-cache add ca-certificates tzdata

# สร้าง non-root user เพื่อความปลอดภัยตาม Best Practice
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# คัดลอก binary มาจาก builder stage
COPY --from=builder /app/bin/api /app/api

# กำหนด Default Environment Variables
ENV PORT=8080 \
    MONGO_URI=mongodb://mongodb:27017 \
    DB_NAME=user_management_db \
    JWT_SECRET=super-secret-key-change-in-production-12345 \
    JWT_EXPIRATION_HOURS=24 \
    USER_COUNT_INTERVAL_SECONDS=10

EXPOSE 8080 50051

ENTRYPOINT ["/app/api"]
