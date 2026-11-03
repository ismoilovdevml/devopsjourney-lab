# Backend developer targets. Run from backend-go/.
.PHONY: help build run test cover vet lint fmt deps docker-build clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-13s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the server binary into bin/
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server

run: ## Run the server (reads .env in this directory)
	go run ./cmd/server

test: ## Run tests with the race detector
	go test -race -count=1 ./...

cover: ## Run tests with coverage report (coverage.out is git-ignored)
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

vet: ## go vet
	go vet ./...

lint: ## golangci-lint
	golangci-lint run ./...

fmt: ## gofmt
	gofmt -s -w .

deps: ## Download and tidy modules
	go mod download
	go mod tidy

docker-build: ## Build the backend image
	docker build --build-arg VERSION=$(VERSION) -t devopslab-backend:$(VERSION) .

clean: ## Remove build artifacts
	rm -rf bin/ coverage.out
