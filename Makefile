.PHONY: help build test run run-grpc docker-up docker-down clean proto swagger

help: ## แสดงคำสั่งที่สามารถใช้งานได้
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## คอมไพล์โปรเจกต์ทั้ง REST API และ gRPC Server
	@echo "Building binaries..."
	go build -o bin/api ./cmd/api
	go build -o bin/grpc ./cmd/grpc
	@echo "Build completed successfully."

test: ## รัน Unit Tests ทั้งหมด
	@echo "Running unit tests with race detection..."
	go test -v -race ./...

run: ## รัน REST API Server ในเครื่อง Local
	go run ./cmd/api

run-grpc: ## รัน gRPC Server ในเครื่อง Local
	go run ./cmd/grpc

proto: ## คอมไพล์ไฟล์ .proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/user.proto

swagger: ## เจนเนอเรตเอกสาร Swagger / OpenAPI
	swag init -g cmd/api/main.go -o docs

docker-up: ## สตาร์ท MongoDB และ API Server ด้วย Docker Compose
	docker-compose up -d --build

docker-down: ## หยุดการทำงานของ Docker Compose และลบ Containers
	docker-compose down

clean: ## ลบไฟล์ที่คอมไพล์แล้ว
	rm -rf bin/
