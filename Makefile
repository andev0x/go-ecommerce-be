# Makefile for E-commerce Microservices

.PHONY: build run test clean docker-build docker-run migrate-up migrate-down

# Build all services
build:
	@echo "Building all services..."
	@go build -o bin/product-service ./cmd/product-service
	@go build -o bin/cart-service ./cmd/cart-service
	@go build -o bin/order-service ./cmd/order-service
	@go build -o bin/delivery-service ./cmd/delivery-service
	@go build -o bin/notification-service ./cmd/notification-service
	@go build -o bin/api-gateway ./cmd/api-gateway

# Run all services in development
run-all:
	@echo "Starting all services..."
	@make run-product &
	@make run-cart &
	@make run-order &
	@make run-delivery &
	@make run-notification &
	@make run-gateway &
	@wait

run-product:
	@echo "Starting Product Service..."
	@go run ./cmd/product-service

run-cart:
	@echo "Starting Cart Service..."
	@go run ./cmd/cart-service

run-order:
	@echo "Starting Order Service..."
	@go run ./cmd/order-service

run-delivery:
	@echo "Starting Delivery Service..."
	@go run ./cmd/delivery-service

run-notification:
	@echo "Starting Notification Service..."
	@go run ./cmd/notification-service

run-gateway:
	@echo "Starting API Gateway..."
	@go run ./cmd/api-gateway

# Testing
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

# Database migrations
migrate-up:
	@echo "Running database migrations..."
	@migrate -path migrations -database "postgres://postgres:password@localhost:5432/ecommerce?sslmode=disable\" up

migrate-down:
	@echo "Rolling back database migrations..."
	@migrate -path migrations -database "postgres://postgres:password@localhost:5432/ecommerce?sslmode=disable\" down

# Docker operations
docker-build:
	@echo "Building Docker images..."
	@docker build -t ecommerce/product-service -f docker/product-service/Dockerfile .
	@docker build -t ecommerce/cart-service -f docker/cart-service/Dockerfile .
	@docker build -t ecommerce/order-service -f docker/order-service/Dockerfile .
	@docker build -t ecommerce/delivery-service -f docker/delivery-service/Dockerfile .
	@docker build -t ecommerce/notification-service -f docker/notification-service/Dockerfile .
	@docker build -t ecommerce/api-gateway -f docker/api-gateway/Dockerfile .

docker-run:
	@echo "Starting services with Docker Compose..."
	@docker-compose up -d

docker-stop:
	@echo "Stopping Docker services..."
	@docker-compose down

# Development setup
setup:
	@echo "Setting up development environment..."
	@go mod tidy
	@docker-compose up -d postgres redis rabbitmq
	@sleep 10
	@make migrate-up

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@rm -f proto/*.pb.go

# Generate protobuf files
.PHONY: proto
proto:
	@echo "Generating protobuf files..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto


# Load testing
load-test:
	@echo "Running load tests..."
	@k6 run scripts/load-test.js

# Security scan
security-scan:
	@echo "Running security scan..."
	@gosec ./...

# Lint code
lint:
	@echo "Running linter..."
	@golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Generate API documentation
docs:
	@echo "Generating API documentation..."
	@swag init -g cmd/api-gateway/main.go