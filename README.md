# Backend Challenge: User Management API & Lottery Search System

โปรเจกต์นี้จัดทำขึ้นสำหรับ Backend Golang Coding Test ประกอบด้วย 2 ส่วนหลัก:

1. **User Management API**: RESTful API และ gRPC Service ที่เขียนด้วย Go (Golang), ใช้ MongoDB ในการจัดเก็บข้อมูล, ระบบพิสูจน์ตัวตนด้วย JWT (HS256), สถาปัตยกรรม Hexagonal Architecture, Background Concurrency Task, และชุด Unit Tests ครบถ้วน
2. **Lottery Search System**: ข้อเสนอการออกแบบสถาปัตยกรรมระบบค้นหาและจัดสรรสลาก 10 ล้านใบ ด้วย Wildcard Pattern Matching แบบ Real-time ป้องกันการได้เลขซ้ำพร้อมกันในระดับ Concurrency (ดูรายละเอียดฉบับเต็มได้ที่ [LOTTERY_SEARCH_DESIGN.md](LOTTERY_SEARCH_DESIGN.md))

---

## 🌟 ฟีเจอร์ที่พัฒนา (Features Implemented)

### User Management API (Part 1)

- [X] **User Entity Model**: กำหนดโครงสร้างผู้ใช้ครบถ้วน (`ID`, `Name`, `Email`, `Password` แบบ Bcrypt Hash, `CreatedAt`)
- [X] **Authentication & Security**:
  - สมัครสมาชิก (Register) พร้อมตรวจสอบความถูกต้องของข้อมูล (Input Validation) และป้องกัน Email ซ้ำ
  - เข้าสู่ระบบ (Login) ตรวจสอบ Password Hash และออก JWT Token (HMAC-SHA256 / `HS256`)
  - Middleware ตรวจสอบ JWT และดึง Claims เข้าสู่ Request Context
- [X] **CRUD Operations**:
  - `POST /api/v1/users` (สร้างผู้ใช้)
  - `GET /api/v1/users` (ดึงรายชื่อผู้ใช้ทั้งหมด)
  - `GET /api/v1/users/{id}` (ค้นหาผู้ใช้ตาม ObjectID)
  - `PUT /api/v1/users/{id}` (แก้ไขชื่อและอีเมล)
  - `DELETE /api/v1/users/{id}` (ลบผู้ใช้)
- [X] **MongoDB Integration**: เชื่อมต่อผ่าน Official Go MongoDB Driver พร้อมสร้าง Unique Index บน Email
- [X] **Logging Middleware**: ดักจับและบันทึก HTTP Method, URL Path, Status Code, IP, และ Execution Time ทุก Request
- [X] **Concurrency Task**: Background Goroutine ตรวจสอบและบันทึกจำนวนผู้ใช้ทั้งหมดในฐานข้อมูลทุกๆ 10 วินาที พร้อมรองรับ Context Cancellation
- [X] **Unit Testing & Mocking**: เขียนเทสด้วย standard `testing` package ของ Go โดยใช้ In-memory Mock Repository ไม่ต้องต่อ Database จริง (ครอบคลุมทั้ง Service Layer และ HTTP Handler Layer)

### Bonus Features (ครบทุกข้อตาม README)

- [X] **Containerization**: มี `Dockerfile` แบบ Multi-stage และ `docker-compose.yml` รองรับการรัน API คู่กับ MongoDB แบบ 1-Command
- [X] **Abstraction & Hexagonal Architecture**: แยก Layer ชัดเจน (Domain, Repository, Service, Transport) ผ่าน Go Interfaces
- [X] **Input Validation**: ตรวจสอบ RFC 5322 Email Format, ความยาวรหัสผ่าน, และ Required Fields
- [X] **Graceful Shutdown**: ดักจับ OS Signals (`SIGINT`, `SIGTERM`) เพื่อปิด HTTP Server, Disconnect MongoDB และหยุด Goroutine อย่างปลอดภัย
- [X] **gRPC Support**: มีไฟล์ `proto/user.proto`, มี gRPC Server สำหรับ `CreateUser` และ `GetUser` พร้อม Secure Token Metadata Interceptor

---

## 📁 โครงสร้างโปรเจกต์ (Project Structure)

```
├── cmd/
│   ├── api/
│   │   └── main.go                # REST API Entry point & Background worker
│   └── grpc/
│       └── main.go                # gRPC Server Entry point (Bonus)
├── internal/
│   ├── config/
│   │   └── config.go              # โหลด Environment Variables
│   ├── domain/
│   │   ├── user.go                # Entity & Repository Interface (Ports)
│   │   └── errors.go              # Domain Error definitions
│   ├── pkg/
│   │   ├── hasher/
│   │   │   └── hasher.go          # Bcrypt hashing utilities
│   │   └── jwt/
│   │       └── jwt.go             # JWT Token Generator & Validator (HS256)
│   ├── repository/
│   │   ├── mongodb/
│   │   │   └── user_repo.go       # MongoDB Adapter (Official Driver)
│   │   └── mock/
│   │       └── user_repo_mock.go  # In-memory Mock Repository สำหรับ Unit Tests
│   ├── service/
│   │   ├── user_service.go        # Business Logic Use Cases
│   │   └── user_service_test.go   # Unit Tests สำหรับ Service
│   ├── transport/
│   │   ├── http/
│   │   │   ├── handler.go         # REST API Handlers
│   │   │   ├── handler_test.go    # HTTP Integration/Unit Tests
│   │   │   ├── middleware.go      # Logging & JWT Auth Middlewares
│   │   │   └── response.go        # Unified JSON Response format
│   │   └── grpc/
│   │       ├── server.go          # gRPC Service Implementation
│   │       └── auth_interceptor.go # gRPC Metadata Token Interceptor
│   └── worker/
│       └── counter.go             # 10s Concurrency Background Goroutine
├── proto/
│   ├── user.proto                 # Protobuf definitions
│   ├── user.pb.go                 # Generated Protobuf Go code
│   └── user_grpc.pb.go            # Generated gRPC Go code
├── Dockerfile                     # Multi-stage Docker build
├── docker-compose.yml             # Docker Compose orchestration
├── Makefile                       # คำสั่งลัด (build, test, run, docker-up)
├── LOTTERY_SEARCH_DESIGN.md       # เอกสารข้อเสนอการออกแบบระบบสลาก 10 ล้านใบ (Part 2)
└── README.md                      # เอกสารแนะนำการใช้งานฉบับนี้
```

---

## 🚀 วิธีการติดตั้งและรันระบบ (Setup & Execution)

### ทางเลือกที่ 1: รันผ่าน Docker Compose (แนะนำ สะดวกที่สุด)

เพียงติดตั้ง Docker และรันคำสั่ง:

```bash
docker-compose up -d --build
```

ระบบจะเปิด:

- [ ] **MongoDB**: `localhost:27017`
- [ ] **User Management API**: `http://localhost:8080`

ตรวจสอบสถานะการทำงาน:

```bash
docker-compose logs -f api
```

หยุดการทำงาน:

```bash
docker-compose down
```

---

### ทางเลือกที่ 2: รันในเครื่อง Local (Local Machine)

#### สิ่งที่จำเป็น (Prerequisites):

- Go 1.24+
- MongoDB รันอยู่ที่ `mongodb://localhost:27017` (หรือรัน `docker run -d -p 27017:27017 --name local-mongo mongo:7.0`)

#### 1. รัน REST API Server:

```bash
go run ./cmd/api
```

หรือใช้ Makefile:

```bash
make run
```

#### 2. รัน gRPC Server (Bonus):

```bash
go run ./cmd/grpc
# หรือ make run-grpc
```

---

## 🧪 การรัน Unit Tests

โปรเจกต์นี้มีชุดทดสอบครอบคลุมทั้ง Unit Tests และ Transport Handler Tests พร้อมตรวจจับ Data Race ด้วย:

```bash
go test -v -race ./...
```

หรือรันผ่าน Makefile:

```bash
make test
```

---

## 🔑 คู่มือการใช้งาน JWT Token (JWT Guide)

### 1. วิธีการทำงานและการสร้าง Token

- Token ถูกสร้างขึ้นเมื่อผู้ใช้ทำการเข้าสู่ระบบสำเร็จผ่าน `POST /api/v1/auth/login`
- ใช้ Algorithm **HMAC-SHA256 (`HS256`)** พร้อมลงนามด้วย Secret Key (กำหนดผ่าน Environment Variable `JWT_SECRET`)
- ข้อมูลใน Payload (Claims) ประกอบด้วย:
  - `user_id`: MongoDB ObjectID ในรูปแบบ Hex string
  - `email`: อีเมลของผู้ใช้
  - `exp`: วันหมดอายุ (ค่าเริ่มต้นคือ 24 ชั่วโมง)
  - `iat`: วันที่ออก Token
  - `iss`: ผู้ออก Token (`backend-challenge-api`)

### 2. วิธีการนำ Token ไปใช้งานใน Protected Endpoints

ให้นำ Token ที่ได้รับจากการ Login ไปใส่ใน HTTP Request Header:

```http
Authorization: Bearer <your_jwt_token_here>
```

หากไม่มี Token หรือ Token หมดอายุ/ไม่ถูกต้อง ระบบจะส่ง HTTP 401 Unauthorized กลับมาทันที

---

## 📡 ตัวอย่าง API Requests & Responses (Sample API Calls)

### 1. Health Check

```bash
curl -s http://localhost:8080/health | jq
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

### 2. สมัครสมาชิกใหม่ (User Registration)

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Somchai Jaidee",
    "email": "somchai@example.com",
    "password": "password123"
  }' | jq
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

### 3. เข้าสู่ระบบ (User Login) เพื่อรับ JWT Token

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "somchai@example.com",
    "password": "password123"
  }' | jq
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

> **ทริค**: เก็บ Token ไว้ในตัวแปร Shell:
>
> ```bash
> TOKEN="<token_ที่ได้จาก_response>"
> ```

---

### 4. ดึงรายชื่อผู้ใช้ทั้งหมด (List Users) [Protected]

```bash
curl -s -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN" | jq
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

### 5. ค้นหาผู้ใช้ตาม ID (Get User by ID) [Protected]

```bash
curl -s -X GET http://localhost:8080/api/v1/users/67da34fb879a957c129486c0 \
  -H "Authorization: Bearer $TOKEN" | jq
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

### 6. อัปเดตข้อมูลผู้ใช้ (Update User) [Protected]

```bash
curl -s -X PUT http://localhost:8080/api/v1/users/67da34fb879a957c129486c0 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Somchai Update",
    "email": "somchai.new@example.com"
  }' | jq
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
  -H "Authorization: Bearer $TOKEN" | jq
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

## ⚙️ การตัดสินใจเชิงสถาปัตยกรรม (Design Decisions & Assumptions)

1. **Hexagonal Architecture (Ports and Adapters)**:
   - แกนกลาง (Domain) ไม่มี dependency ผูกติดกับ Database หรือ Framework ภายนอก
   - `domain.UserRepository` เป็น Interface (Port) และ `mongodb.MongoUserRepository` เป็น Implementation (Adapter)
   - ทำให้สามารถสลับฐานข้อมูล หรือจำลอง Mock สำหรับ Unit Test ได้ 100% โดยไม่ต้องพึ่ง external service
2. **ความปลอดภัยของรหัสผ่าน (Password Security)**:
   - แฮชรหัสผ่านด้วย `bcrypt` (Default Cost 10) ที่มี Salt ฝังในตัว ป้องกันการโจมตีแบบ Rainbow Table
   - ซ่อนฟิลด์รหัสผ่านใน JSON Response เสมอด้วย Tag `json:"-"`
3. **ความเสถียรของ Database**:
   - สร้าง Unique Index บนฟิลด์ `email` ตั้งแต่จังหวะเริ่มต้นทำงาน เพื่อรับประกันความไม่ซ้ำกันของข้อมูลแม้จะเกิด Race Condition ระดับแอปพลิเคชัน
4. **Concurrency Background Task**:
   - รันใน Goroutine แยกและใช้ `time.Ticker` ความถี่ 10 วินาทีตาม Requirement 6
   - ผูก `context.Context` เพื่อให้หยุดทำงานทันทีที่มีสัญญาณ Graceful Shutdown ไม่เกิด Goroutine Leak
5. **การจัดการ Graceful Shutdown**:
   - จัดการตัดการเชื่อมต่อแบบเป็นลำดับ: หยุดรับ Request ใหม่ $\rightarrow$ ยกเลิก Background Worker $\rightarrow$ ปิดการเชื่อมต่อ MongoDB

---

## 🎟️ ส่วนที่ 2: ระบบค้นหาสลาก (Lottery Search System Design)

สำหรับข้อกำหนดส่วนที่ 2 (แบบร่างสถาปัตยกรรมระดับระบบ ไม่เขียนโค้ด):

- รองรับข้อมูล **10 ล้านใบ**
- รองรับการค้นหา Wildcard Pattern เช่น `****23`, `1****5`, `123***`
- **ระบบป้องกันการส่งเลขเดียวกันให้ผู้ใช้หลายคนพร้อมกัน** (Zero Double Allocation) ผ่าน Atomic Redis Lua Two-Phase Lease
- การเลือก Database ที่เหมาะสมในระดับ Production (Redis Cluster + Partitioned PostgreSQL 16)

👉 **อ่านรายละเอียดการออกแบบทางสถาปัตยกรรมอย่างละเอียดได้ที่:**
**[LOTTERY_SEARCH_DESIGN.md](LOTTERY_SEARCH_DESIGN.md)**
