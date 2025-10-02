.PHONY: build run test docker-build docker-run clean deps

# Build the application
build:
	go build -o bin/server ./cmd/server

# Run the application
run:
	go run ./cmd/server/main.go

# Run tests
test:
	go test -v -cover ./...

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build Docker image
docker-build:
	docker build -t devopslab-backend:latest .

# Run with Docker
docker-run:
	docker run -p 8080:8080 --env-file .env devopslab-backend:latest

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Development with hot reload
dev:
	go run github.com/cosmtrek/air@latest

# Generate swagger docs
swagger:
	swag init -g cmd/server/main.go

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# All
all: deps fmt build test
