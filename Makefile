.PHONY: help build run stop clean logs test docker-build docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the Go application
	go build -o bin/server ./cmd/server

run: ## Run the application locally (requires MySQL running)
	go run ./cmd/server/main.go

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -rf bin/
	go clean

docker-build: ## Build Docker image
	docker-compose build

docker-up: ## Start the application with Docker Compose
	docker-compose up -d

docker-down: ## Stop and remove Docker containers
	docker-compose down

docker-logs: ## Show Docker logs
	docker-compose logs -f

docker-restart: ## Restart Docker containers
	docker-compose restart

docker-clean: ## Remove Docker containers and volumes
	docker-compose down -v

deps: ## Download Go dependencies
	go mod download
	go mod tidy

fmt: ## Format Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: fmt vet ## Run linters

migration-status: ## Check migration status
	@echo "Migrations are auto-applied on startup via GORM AutoMigrate"
