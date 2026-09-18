# User Management API & Lottery Search System

เอกสารประกอบการส่งแบบทดสอบตำแหน่ง Backend Engineer ครอบคลุมการพัฒนาระบบ User Management API ด้วยภาษา Go และข้อเสนอการออกแบบเชิงสถาปัตยกรรมสำหรับระบบค้นหาสลากกินแบ่ง (Lottery Search System)

โปรเจกต์ประกอบด้วย 2 ส่วนหลัก:

1. **User Management API**: RESTful API และ gRPC Service พัฒนาด้วย Golang, ใช้ MongoDB ในการจัดเก็บข้อมูล, ยืนยันตัวตนด้วย JWT (HS256), วางโครงสร้างแบบ Hexagonal Architecture (Ports and Adapters), มี Background Concurrency Task และชุด Unit Tests
2. **Lottery Search System**: ข้อเสนอการออกแบบสถาปัตยกรรมระบบค้นหาและจัดสรรสลาก 10 ล้านใบ ด้วย Wildcard Pattern Matching แบบเรียลไทม์ พร้อมกลไกป้องกันการเลือกเลขซ้ำซ้อนในสภาวะ Concurrency สูง (เอกสารฉบับเต็ม: [ภาษาไทย (LOTTERY_SEARCH_DESIGN_TH.md)](LOTTERY_SEARCH_DESIGN_TH.md) | [English (LOTTERY_SEARCH_DESIGN.md)](LOTTERY_SEARCH_DESIGN.md))

---

## สรุปข้อกำหนดและการพัฒนา (Implementation Overview)

### 1. User Management API (Part 1)

- **User Entity Model**: กำหนดโครงสร้าง Entity ครบถ้วนตามโจทย์ (`ID`, `Name`, `Email`, `Password` แบบ Bcrypt Hash, `CreatedAt`)
- **Authentication & Security**:
  - ระบบสมัครสมาชิก (Register) พร้อม Input Validation ตรวจสอบรูปแบบอีเมลตามมาตรฐาน RFC 5322 และความยาวรหัสผ่าน
  - ระบบเข้าสู่ระบบ (Login) ตรวจสอบ Password Hash และออก JWT Token ลงนามด้วย HMAC-SHA256 (`HS256`)
  - Middleware ตรวจสอบ JWT พร้อมสกัด User Claims เข้าสู่ Request Context
- **CRUD Operations**:
  - `POST /api/v1/users`: สร้างผู้ใช้ใหม่
  - `GET /api/v1/users`: ดึงรายชื่อผู้ใช้ทั้งหมด
  - `GET /api/v1/users/{id}`: ค้นหาผู้ใช้ตาม ObjectID
  - `PUT /api/v1/users/{id}`: ปรับปรุงข้อมูลชื่อและอีเมล
  - `DELETE /api/v1/users/{id}`: ลบผู้ใช้ตาม ID
- **MongoDB Integration**: พัฒนาด้วย Official Go MongoDB Driver พร้อมสร้าง Unique Index บนฟิลด์ `email`
- **Logging Middleware**: บันทึกข้อมูล HTTP Method, Request Path, HTTP Status Code, Client IP และ Execution Duration ทุกคำขอ
- **Concurrency Task**: Background Goroutine ตรวจสอบและบันทึกจำนวนผู้ใช้ในฐานข้อมูลทุกๆ 10 วินาที พร้อมรองรับ Context Cancellation
- **Testing**: ชุด Unit Tests ด้วย Go Standard `testing` package ร่วมกับ In-Memory Mock Repository โดยไม่ต้องพึ่งพา External Database จริง

### 2. ข้อกำหนดเพิ่มเติม (Bonus Implementations)

- **Containerization**: พัฒนา `Dockerfile` แบบ Multi-Stage Build (Minimal Runtime) และ `docker-compose.yml` รองรับการรัน API คู่กับ MongoDB
- **Hexagonal Architecture**: แยก Layer อย่างอิสระ (Domain, Repository, Service, Transport) ผ่าน Go Interfaces เพื่อความยืดหยุ่นในการขยายระบบ
- **Input Validation**: ตรวจสอบความถูกต้องของข้อมูลทุก Endpoint ป้องกันข้อผิดพลาดตั้งแต่ Transport Layer
- **Graceful Shutdown**: ดักจับ OS Signals (`SIGINT`, `SIGTERM`) เพื่อคืน Resource, ปิด HTTP Listener, ตัดการเชื่อมต่อ MongoDB และหยุด Background Worker อย่างปลอดภัย
- **gRPC Support**: จัดทำไฟล์ Protobuf (`proto/user.proto`) พร้อมพัฒนา gRPC Server รองรับการจัดการข้อมูลผู้ใช้ครบถ้วนตามหลัก CRUD (`CreateUser`, `GetUser`, `ListUsers`, `UpdateUser`, `DeleteUser`) ร่วมกับ Metadata Authentication Interceptor ป้องกันการเข้าถึงโดยไม่ได้รับอนุญาต
- **API Documentation (Swagger UI)**: เอกสาร API มาตรฐาน OpenAPI 2.0 พร้อมหน้าเว็บ Swagger UI แบบ Interactive เข้าถึงได้ผ่าน Browser ที่ `http://localhost:8080/swagger` พร้อมปุ่ม Authorize สำหรับใส่ Bearer Token
- **Web Dashboard**: หน้าเว็บสำหรับทดสอบ User Management API เข้าใช้งานได้ทันทีที่ `http://localhost:8080/` โดยฝัง Assets ทั้งหมดไว้ใน Go Binary (`//go:embed`) ไม่ต้องติดตั้ง Node.js หรือเปิดพอร์ตเพิ่ม (รองรับ Authentication, จัดการ CRUD ผู้ใช้ และแสดงสถานะระบบ)

---

## โครงสร้างโปรเจกต์ (Project Structure)

```
├── cmd/
│   ├── api/
│   │   └── main.go                # REST API Entry Point & Background Worker
│   └── grpc/
│       └── main.go                # gRPC Server Entry Point
├── docs/                          # Swagger / OpenAPI Generated Files
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── internal/
│   ├── config/
│   │   └── config.go              # การโหลด Environment Variables
│   ├── domain/
│   │   ├── user.go                # Entity และ Repository Interfaces (Ports)
│   │   └── errors.go              # นิยาม Domain Errors
│   ├── pkg/
│   │   ├── hasher/
│   │   │   └── hasher.go          # ยูทิลิตี้การแฮชรหัสผ่านด้วย Bcrypt
│   │   └── jwt/
│   │       └── jwt.go             # การสร้างและตรวจสอบ JWT Token (HS256)
│   ├── repository/
│   │   ├── mongodb/
│   │   │   └── user_repo.go       # MongoDB Repository Implementation (Adapter)
│   │   └── mock/
│   │       └── user_repo_mock.go  # In-memory Mock Repository สำหรับ Unit Tests
│   ├── service/
│   │   ├── user_service.go        # Business Logic Use Cases
│   │   └── user_service_test.go   # Unit Tests สำหรับ Service Layer
│   ├── transport/
│   │   ├── http/
│   │   │   ├── handler.go         # HTTP Handlers (พร้อม Swagger Annotations)
│   │   │   ├── handler_test.go    # HTTP Integration/Route Tests
│   │   │   ├── middleware.go      # Logging, CORS และ JWT Middlewares
│   │   │   └── response.go        # รูปแบบ Standardized JSON Response
│   │   └── grpc/
│   │       ├── server.go          # gRPC Server Implementation (CRUD เต็มรูปแบบ)
│   │       ├── server_test.go     # gRPC Unit Tests
│   │       └── auth_interceptor.go # gRPC Metadata Token Interceptor
│   └── worker/
│       └── counter.go             # Background Goroutine รันนับจำนวนผู้ใช้ทุก 10 วินาที
├── proto/
│   ├── user.proto                 # นิยาม Protocol Buffers
│   ├── user.pb.go                 # Generated Protobuf Go Code
│   └── user_grpc.pb.go            # Generated gRPC Go Code
├── Dockerfile                     # Multi-Stage Build Dockerfile
├── docker-compose.yml             # Orchestration สำหรับ API และ MongoDB
├── Makefile                       # คำสั่ง Build, Test, Proto, Swagger และ Run
├── LOTTERY_SEARCH_DESIGN.md       # ข้อเสนอการออกแบบระบบค้นหาสลาก (English Version)
├── LOTTERY_SEARCH_DESIGN_TH.md    # ข้อเสนอการออกแบบระบบค้นหาสลาก (ฉบับภาษาไทย)
└── README.md                      # เอกสารแนะนำการติดตั้งและการใช้งาน
```

---

## การติดตั้งและการเริ่มใช้งาน (Setup & Execution)

### ทางเลือกที่ 1: รันด้วย Docker Compose (แนะนำ)

สั่งเปิดระบบทั้ง MongoDB และ API ด้วยคำสั่งเดียว:

```bash
docker-compose up -d --build
```

- **HTTP REST API**: `http://localhost:8080`
- **Swagger UI**: `http://localhost:8080/swagger`
- **gRPC Server**: `localhost:50051`
- **MongoDB**: `localhost:27017`

ตรวจสอบการทำงานของระบบ:

```bash
docker-compose logs -f api
```

หยุดการทำงานของ Container:

```bash
docker-compose down
```

---

### ทางเลือกที่ 2: รันบนเครื่อง Local

#### ข้อกำหนดเบื้องต้น (Prerequisites):

- Go 1.24+
- MongoDB instance เปิดทำงานอยู่ที่ `mongodb://localhost:27017`

#### 1. สตาร์ท REST API Server:

```bash
go run ./cmd/api
```

หรือใช้คำสั่งผ่าน Makefile:

```bash
make run
```

#### 2. สตาร์ท gRPC Server (ทางเลือก):

```bash
go run ./cmd/grpc
```

หรือ:

```bash
make run-grpc
```

---

## การทดสอบระบบ (Testing)

โปรเจกต์ประกอบด้วยชุด Unit Tests สำหรับ Service Layer และ Integration Tests สำหรับ HTTP Handler พร้อมเปิด Data Race Detection:

```bash
go test -v -race ./...
```

หรือรันผ่าน Makefile:

```bash
make test
```

ผลการทดสอบ:

```text
=== RUN   TestRegister_Success
--- PASS: TestRegister_Success (0.07s)
=== RUN   TestRegister_DuplicateEmail
--- PASS: TestRegister_DuplicateEmail (0.05s)
=== RUN   TestRegister_ValidationErrors
--- PASS: TestRegister_ValidationErrors (0.00s)
=== RUN   TestLogin_Success
--- PASS: TestLogin_Success (0.10s)
=== RUN   TestLogin_InvalidCredentials
--- PASS: TestLogin_InvalidCredentials (0.10s)
=== RUN   TestGetUserByID
--- PASS: TestGetUserByID (0.05s)
=== RUN   TestListUsers
--- PASS: TestListUsers (0.10s)
=== RUN   TestUpdateUser
--- PASS: TestUpdateUser (0.05s)
=== RUN   TestDeleteUser
--- PASS: TestDeleteUser (0.05s)
=== RUN   TestCountUsers
--- PASS: TestCountUsers (0.05s)
PASS
ok      backend-challenge/internal/service
=== RUN   TestGRPC_CreateUser
--- PASS: TestGRPC_CreateUser (0.06s)
=== RUN   TestGRPC_GetUser
--- PASS: TestGRPC_GetUser (0.05s)
=== RUN   TestGRPC_ListUsers
--- PASS: TestGRPC_ListUsers (0.10s)
=== RUN   TestGRPC_UpdateUser
--- PASS: TestGRPC_UpdateUser (0.05s)
=== RUN   TestGRPC_DeleteUser
--- PASS: TestGRPC_DeleteUser (0.05s)
PASS
ok      backend-challenge/internal/transport/grpc
=== RUN   TestHealthCheckRoute
--- PASS: TestHealthCheckRoute (0.00s)
=== RUN   TestAuthRoutes_RegisterAndLogin
--- PASS: TestAuthRoutes_RegisterAndLogin (0.10s)
```

---

## หน้าเว็บสำหรับทดสอบระบบ (Web Dashboard)

สำหรับทดสอบการทำงานของ User Management API ผ่านหน้าเว็บ สามารถเปิดบราวเซอร์เข้าสู่หน้าเว็บได้ทันทีที่:

👉 **URL: [http://localhost:8080/](http://localhost:8080/)**

*(ไฟล์หน้าเว็บทั้งหมดถูกคอมไพล์ฝังใน Go Binary ผ่าน `//go:embed` ไม่ต้องติดตั้ง Node.js หรือเปิดพอร์ตเพิ่ม)*

### ฟังก์ชันบนหน้าเว็บ:
1. **System Status**: แสดงสถานะเรียลไทม์ของ REST API (`:8080`), gRPC (`:50051`), MongoDB และ Goroutine Worker
2. **Authentication**: สมัครสมาชิก (Sign Up) และเข้าสู่ระบบ (Sign In) รับ JWT Token อัตโนมัติ
3. **User Management (CRUD)**: ค้นหา, เพิ่มผู้ใช้ใหม่, แก้ไขข้อมูล และลบผู้ใช้
4. **Request Inspector**: แสดงคำสั่ง `curl` เทียบเท่าของแต่ละคำขอเพื่อนำไปทดสอบต่อใน Terminal ได้ทันที
5. **Swagger Documentation**: ปุ่มลัดเปิดหน้า Swagger UI (`/swagger`)

---

## ข้อมูลจำเพาะของ JWT Authentication

### 1. โครงสร้างและการสร้าง Token

- Token ออกให้เมื่อผู้ใช้ยืนยันตัวตนสำเร็จผ่าน `POST /api/v1/auth/login`
- Algorithm: **HMAC-SHA256 (`HS256`)** ลงนามด้วย `JWT_SECRET`
- ข้อมูลใน Claims:
  - `user_id`: MongoDB ObjectID ในรูปแบบ Hex string
  - `email`: อีเมลของผู้ใช้
  - `exp`: วันหมดอายุของ Token (Default: 24 ชั่วโมง)
  - `iat`: เวลาที่ออก Token
  - `iss`: ผู้ออก Token (`backend-challenge-api`)

### 2. การเรียกใช้ Protected Endpoints

นำ Token แนบไปกับ HTTP Request Header:

```http
Authorization: Bearer <JWT_TOKEN>
```

กรณีไม่มี Header หรือ Token ไม่ถูกต้อง ระบบจะส่งกลับรหัส `401 Unauthorized`

---

## ตัวอย่างการเรียกใช้งาน API (API Specification & Samples)

> สามารถเปิดทดสอบและดู Schema แบบ Interactive ผ่าน **Swagger UI** ได้โดยตรงที่: **`http://localhost:8080/swagger`** (พร้อมปุ่ม Authorize สำหรับทดสอบ Protected Endpoints ด้วย JWT Token)

### 1. Health Check

```bash
curl -s http://localhost:8080/health
```

**Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "status": "UP"
  },
  "message": "server is healthy"
}
```

---

### 2. สมัครสมาชิก (User Registration)

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Somchai Jaidee",
    "email": "somchai@example.com",
    "password": "password123"
  }'
```

**Response (201 Created):**

```json
{
  "success": true,
  "data": {
    "id": "67da34fb879a957c129486c0",
    "name": "Somchai Jaidee",
    "email": "somchai@example.com",
    "created_at": "2026-09-17T02:30:00Z"
  },
  "message": "user registered successfully"
}
```

---

### 3. เข้าสู่ระบบ (User Login)

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "somchai@example.com",
    "password": "password123"
  }'
```

**Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "67da34fb879a957c129486c0",
      "name": "Somchai Jaidee",
      "email": "somchai@example.com",
      "created_at": "2026-09-17T02:30:00Z"
    }
  },
  "message": "login successful"
}
```

---

### 4. ดึงรายชื่อผู้ใช้ทั้งหมด (List Users) [Protected]

```bash
curl -s -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN"
```

**Response (200 OK):**

```json
{
  "success": true,
  "data": [
    {
      "id": "67da34fb879a957c129486c0",
      "name": "Somchai Jaidee",
      "email": "somchai@example.com",
      "created_at": "2026-09-17T02:30:00Z"
    }
  ],
  "message": "users retrieved successfully"
}
```

---

### 5. ดึงข้อมูลผู้ใช้ตาม ID (Get User by ID) [Protected]

```bash
curl -s -X GET http://localhost:8080/api/v1/users/67da34fb879a957c129486c0 \
  -H "Authorization: Bearer $TOKEN"
```

**Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "id": "67da34fb879a957c129486c0",
    "name": "Somchai Jaidee",
    "email": "somchai@example.com",
    "created_at": "2026-09-17T02:30:00Z"
  },
  "message": "user retrieved successfully"
}
```

---

### 6. ปรับปรุงข้อมูลผู้ใช้ (Update User) [Protected]

```bash
curl -s -X PUT http://localhost:8080/api/v1/users/67da34fb879a957c129486c0 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Somchai Update",
    "email": "somchai.new@example.com"
  }'
```

**Response (200 OK):**

```json
{
  "success": true,
  "data": {
    "id": "67da34fb879a957c129486c0",
    "name": "Somchai Update",
    "email": "somchai.new@example.com",
    "created_at": "2026-09-17T02:30:00Z"
  },
  "message": "user updated successfully"
}
```

---

### 7. ลบผู้ใช้ (Delete User) [Protected]

```bash
curl -s -X DELETE http://localhost:8080/api/v1/users/67da34fb879a957c129486c0 \
  -H "Authorization: Bearer $TOKEN"
```

**Response (200 OK):**

```json
{
  "success": true,
  "data": null,
  "message": "user deleted successfully"
}
```

---

## การทดสอบ gRPC API (gRPC Testing Guide)

ระบบเปิดให้บริการ gRPC Server บนพอร์ต **`50051`** ควบคู่ไปกับ REST API พร้อมเปิดใช้งาน **gRPC Server Reflection** ทำให้สามารถใช้เครื่องมือทดสอบ เช่น `grpcurl` หรือ Postman (gRPC Mode) เรียกใช้งานและสำรวจ Schema ได้ทันทีโดยไม่ต้องโหลดไฟล์ `.proto` ด้วยตนเอง

### 1. เครื่องมือทดสอบ (Installation)

ติดตั้ง `grpcurl` ผ่าน Homebrew:

```bash
brew install grpcurl
```

### 2. ตรวจสอบ Services และ Methods (Service Discovery)

```bash
# ตรวจสอบ Services ที่เปิดให้บริการ
grpcurl -plaintext localhost:50051 list
```

**ผลลัพธ์:**

```text
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
user.UserService
```

```bash
# ตรวจสอบ Methods ภายใต้ user.UserService
grpcurl -plaintext localhost:50051 list user.UserService
```

**ผลลัพธ์:**

```text
user.UserService.CreateUser
user.UserService.DeleteUser
user.UserService.GetUser
user.UserService.ListUsers
user.UserService.UpdateUser
```

---

### 3. จุดที่ 1: การสร้างผู้ใช้ใหม่ (CreateUser - C) [Public]

Endpoint สำหรับสมัครสมาชิก ไม่ต้องแนบ JWT Token:

```bash
grpcurl -plaintext \
  -d '{"name": "Somchai Jaidee", "email": "somchai_grpc@example.com", "password": "password123"}' \
  localhost:50051 user.UserService/CreateUser
```

**ผลลัพธ์ (Response):**

```json
{
  "id": "6aacb46c71edc7877f277dae",
  "name": "Somchai Jaidee",
  "email": "somchai_grpc@example.com",
  "createdAt": "2026-09-18T03:47:56Z"
}
```

---

### 4. จุดที่ 2: การดึงข้อมูลผู้ใช้ (GetUser & ListUsers - R) [Protected]

> **หมายเหตุ:** สำหรับ Endpoint ที่ได้รับการปกป้อง ต้องเข้าสู่ระบบผ่าน REST API (`POST /api/v1/auth/login`) เพื่อรับ JWT Token ก่อน แล้วนำมาเก็บในตัวแปร Environment:
>
> ```bash
> export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
> ```

#### 4.1 ดึงข้อมูลผู้ใช้ตาม ID (GetUser)

แนบ Token ผ่าน Metadata Header `authorization`:

```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{"id": "6aacb46c71edc7877f277dae"}' \
  localhost:50051 user.UserService/GetUser
```

**ผลลัพธ์ (Response):**

```json
{
  "id": "6aacb46c71edc7877f277dae",
  "name": "Somchai Jaidee",
  "email": "somchai_grpc@example.com",
  "createdAt": "2026-09-18T03:47:56Z"
}
```

#### 4.2 ดึงรายชื่อผู้ใช้ทั้งหมด (ListUsers)

```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  localhost:50051 user.UserService/ListUsers
```

**ผลลัพธ์ (Response):**

```json
{
  "users": [
    {
      "id": "6aacb46c71edc7877f277dae",
      "name": "Somchai Jaidee",
      "email": "somchai_grpc@example.com",
      "createdAt": "2026-09-18T03:47:56Z"
    }
  ]
}
```

---

### 5. จุดที่ 3: การปรับปรุงข้อมูลผู้ใช้ (UpdateUser - U) [Protected]

รองรับ Partial Update (สามารถอัปเดตเฉพาะ `name`, เฉพาะ `email` หรือทั้งสองฟิลด์พร้อมกันได้):

```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{
    "id": "6aacb46c71edc7877f277dae",
    "name": "Somchai Pro",
    "email": "somchai_updated@example.com"
  }' \
  localhost:50051 user.UserService/UpdateUser
```

**ผลลัพธ์ (Response):**

```json
{
  "id": "6aacb46c71edc7877f277dae",
  "name": "Somchai Pro",
  "email": "somchai_updated@example.com",
  "createdAt": "2026-09-18T03:47:56Z"
}
```

---

### 6. จุดที่ 4: การลบผู้ใช้ (DeleteUser - D) [Protected]

ลบผู้ใช้ตาม ObjectID:

```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{"id": "6aacb46c71edc7877f277dae"}' \
  localhost:50051 user.UserService/DeleteUser
```

**ผลลัพธ์ (Response):**

```json
{
  "success": true,
  "message": "user deleted successfully"
}
```

---

### 7. การตรวจสอบข้อผิดพลาดและความปลอดภัย (Error Handling & Security)

gRPC Server ส่งกลับมาตรฐาน Status Codes ตามข้อกำหนดของ gRPC:

| กรณีทดสอบ               | สถานะ gRPC Code     | คำสั่งทดสอบ                                                                                                                    | ผลลัพธ์ที่ได้รับ                                               |
| :------------------------------- | :----------------------- | :---------------------------------------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------- |
| **ไม่มี Token**       | `Unauthenticated (16)` | `grpcurl -plaintext -d '{"id": "..."}' localhost:50051 user.UserService/GetUser`                                                        | `ERROR: Code: Unauthenticated, Message: authorization token is not provided` |
| **ไม่พบข้อมูล** | `NotFound (5)`         | `grpcurl -plaintext -H "authorization: Bearer $TOKEN" -d '{"id": "6aacb46c71edc7877f277dae"}' localhost:50051 user.UserService/GetUser` | `ERROR: Code: NotFound, Message: user not found`                             |
| **รูปแบบ ID ผิด** | `InvalidArgument (3)`  | `grpcurl -plaintext -H "authorization: Bearer $TOKEN" -d '{"id": "invalid-id"}' localhost:50051 user.UserService/GetUser`               | `ERROR: Code: InvalidArgument, Message: invalid user id format`              |
| **อีเมลซ้ำ**       | `AlreadyExists (6)`    | `grpcurl -plaintext -d '{"name": "...", "email": "somchai_grpc@example.com", ...}' localhost:50051 user.UserService/CreateUser`         | `ERROR: Code: AlreadyExists, Message: email already exists`                  |

---

### 8. การทดสอบผ่าน Postman (Postman gRPC Testing)

Postman (v10 ขึ้นไป) รองรับการยิงคำขอ gRPC ได้โดยตรงผ่านระบบ Server Reflection:

1. **เปิด Request แบบ gRPC**:
   - กดปุ่ม **New** -> เลือก **gRPC** (หรือกดปุ่ม `+` เพิ่มแท็บ แล้วเปลี่ยน Protocol เป็น `gRPC`)
2. **ระบุ URL ของ Server**:
   - ใส่ `localhost:50051` ในช่อง Server URL
3. **เลือก Service และ Method (ผ่าน Server Reflection)**:
   - คลิกที่ช่องเลือก Method ด้านล่าง URL
   - เลือก **Use Server Reflection** (ระบบจะดึง `user.UserService` มาให้โดยอัตโนมัติ)
   - เลือก Method ที่ต้องการทดสอบ เช่น `CreateUser`, `GetUser`, `UpdateUser`, `DeleteUser`, หรือ `ListUsers`
4. **การส่ง Request Body (แท็บ Message)**:
   - เลือกแท็บ **Message** แล้วใส่ข้อมูล JSON ตามต้องการ (เช่น ข้อมูล `name`, `email`, `password`)
5. **การแนบ Token ยืนยันตัวตน (แท็บ Metadata)**:
   - สำหรับ Endpoint ที่ต้องการสิทธิ์ (`GetUser`, `UpdateUser`, `DeleteUser`, `ListUsers`):
   - เลือกแท็บ **Metadata** ด้านข้างแท็บ Message
   - เพิ่ม Key: `authorization`
   - ใส่ Value: `Bearer <JWT_TOKEN>` (Token ที่ได้จากการ Login ผ่าน REST API)
6. **ส่งคำขอ**:
   - กดปุ่ม **Invoke** ผลลัพธ์และ Status Code จะแสดงที่หน้าต่างด้านล่างทันที

---

## การตัดสินใจเชิงสถาปัตยกรรม (Architectural Decisions)

1. **Hexagonal Architecture (Ports and Adapters)**:
   - แกนกลาง (Domain) ไม่มี dependency ผูกติดกับ Database หรือ Framework ภายนอก
   - ใช้ `domain.UserRepository` เป็น Interface (Port) และ `mongodb.MongoUserRepository` เป็น Implementation (Adapter)
   - ทำให้สามารถเปลี่ยนฐานข้อมูล หรือจำลอง Mock สำหรับ Unit Test ได้ 100% โดยไม่ต้องพึ่ง external service
2. **Password Security**:
   - แฮชรหัสผ่านด้วย `bcrypt` (Default Cost 10) ที่มี Salt ภายในตัว ป้องกันการโจมตีแบบ Rainbow Table
   - ซ่อนฟิลด์รหัสผ่านใน JSON Response เสมอด้วย Struct Tag `json:"-"`
3. **Database Constraints**:
   - สร้าง Unique Index บนฟิลด์ `email` ตั้งแต่จังหวะเริ่มต้น Application เพื่อรับประกันความไม่ซ้ำกันของข้อมูลที่ระดับ Database Engine
4. **Concurrency Task Isolation**:
   - รันใน Goroutine แยกและใช้ `time.Ticker` ความถี่ 10 วินาทีตาม Requirement 6
   - ผูก `context.Context` เพื่อให้หยุดทำงานทันทีเมื่อได้รับสัญญาณ Graceful Shutdown ป้องกันปัญหารั่วไหลของ Goroutine (Goroutine Leak)
5. **Graceful Termination**:
   - จัดการตัดการเชื่อมต่อแบบเป็นลำดับ: หยุดรับ Request ใหม่ -> ยกเลิก Context ของ Background Worker -> ปิดการเชื่อมต่อ MongoDB Client

---

## ข้อเสนอการออกแบบระบบค้นหาสลาก (Lottery Search System Design)

สำหรับข้อกำหนดส่วนที่ 2 (แบบร่างสถาปัตยกรรมระบบ ไม่มีการเขียนโค้ด):

- รองรับข้อมูล **10,000,000 ใบ**
- รองรับการค้นหา Wildcard Pattern เช่น `****23`, `1****5`, `123***`
- รับประกันการไม่จัดสรรสลากใบเดียวกันให้ผู้ใช้หลายคนพร้อมกัน (**Zero Duplicate Allocation**) ด้วย Atomic Two-Phase Lease ผ่าน Redis Lua Script
- การเลือกเทคโนโลยีฐานข้อมูลระดับ Production: Redis Cluster (In-Memory Search & Indexing) ร่วมกับ Partitioned PostgreSQL 16 (Transactional Audit Log)

รายละเอียดการออกแบบทางสถาปัตยกรรมฉบับเต็ม:

- **ฉบับภาษาไทย**: [LOTTERY_SEARCH_DESIGN_TH.md](LOTTERY_SEARCH_DESIGN_TH.md)
- **English Version**: [LOTTERY_SEARCH_DESIGN.md](LOTTERY_SEARCH_DESIGN.md)
