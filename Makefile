# Easy Orders Backend Makefile

.PHONY: help build run test clean docker-up docker-down docker-build dev migrate

# Default target
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application locally"
	@echo "  test          - Run tests"
	@echo "  test-race     - Run tests with race detection"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-up     - Start all services with Docker Compose"
	@echo "  docker-down   - Stop all services"
	@echo "  docker-build  - Build Docker image"
	@echo "  dev           - Start development environment with hot reload"
	@echo "  env-setup     - Set up environment configuration from template"
	@echo "  env-check     - Validate docker-compose configuration"
	@echo "  compile-check - Check Go compilation"
	@echo "  validate      - Run full validation (env + compile)"
	@echo "  migrate-up    - Run database migrations"
	@echo "  migrate-down  - Rollback database migrations"
	@echo "  seed          - Seed database with test data"
	@echo "  lint          - Run linters"
	@echo "  fmt           - Format code"

# Build the application
build:
	go build -o bin/server cmd/server/main.go

# Run the application locally
run:
	go run cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Run tests with race detection
test-race:
	go test -race -v ./...

# Clean build artifacts
clean:
	rm -rf bin/ tmp/

# Start all services with Docker Compose (uses .env file)
docker-up:
	@echo "Starting services with configuration from .env file..."
	docker-compose up -d

# Stop all services
docker-down:
	docker-compose down

# Build Docker image
docker-build:
	docker build -t easy-orders-backend .

# Start development environment with hot reload (uses .env file)
dev:
	@echo "Starting development environment with configuration from .env file..."
	docker-compose -f docker-compose.dev.yml up

# Run database migrations (placeholder - will implement later)
migrate-up:
	@echo "Database migrations will be implemented with GORM"

migrate-down:
	@echo "Database rollback will be implemented with GORM"

# Seed database with test data (placeholder)
seed:
	@echo "Database seeding will be implemented later"

# Run linters
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Install development tools
install-tools:
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Generate swagger docs (will implement later)
swagger:
	swag init -g cmd/server/main.go -o docs/

# Database shell
db-shell:
	docker-compose exec postgres psql -U postgres -d easy_orders

# Redis shell  
redis-shell:
	docker-compose exec redis redis-cli

# View logs
logs:
	docker-compose logs -f app

# View development logs
logs-dev:
	docker-compose -f docker-compose.dev.yml logs -f app-dev

# Set up environment configuration
env-setup:
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example..."; \
		cp .env.example .env; \
		echo "✅ .env file created. Please edit it with your configuration."; \
	else \
		echo "✅ .env file already exists."; \
	fi

# Validate environment configuration
env-check:
	@echo "Validating docker-compose configuration..."
	docker-compose config > /dev/null && echo "✅ Configuration is valid"

# Check compilation without building
compile-check:
	@echo "Checking Go compilation..."
	go build -o /dev/null cmd/server/main.go && echo "✅ Code compiles successfully"

# Validate entire setup
validate:
	@echo "Running full validation..."
	@make env-check
	@make compile-check
	@echo "✅ All validations passed"