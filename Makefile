# Easy Orders Backend Makefile

.PHONY: help build run test clean docker-up docker-down docker-build dev debug debug-down debug-logs migrate

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
	@echo "  debug         - Start debug environment with Delve for GoLand"
	@echo "  debug-down    - Stop debug environment"
	@echo "  debug-logs    - View debug container logs"
	@echo "  env-setup     - Set up environment configuration from template"
	@echo "  env-check     - Validate docker-compose configuration"
	@echo "  compile-check - Check Go compilation"
	@echo "  validate      - Run full validation (env + compile)"
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
	docker-compose -f docker-compose.dev.yml up -d

# Stop development environment
dev-down:
	@echo "Stopping development environment..."
	docker-compose -f docker-compose.dev.yml down

# Start debug environment with Delve debugger (uses .env file)
debug:
	@echo "Starting debug environment with Delve debugger..."
	@echo "After containers start, use 'Debug in Docker' run configuration in GoLand"
	docker-compose -f docker-compose.debug.yml up -d
	@echo "Waiting for debugger to be ready..."
	@sleep 3
	@echo "✅ Debug environment ready. Delve listening on localhost:2345"
	@echo "📝 Click 'Debug' in GoLand to attach the debugger"

# Stop debug environment
debug-down:
	@echo "Stopping debug environment..."
	docker-compose -f docker-compose.debug.yml down

# View debug container logs
debug-logs:
	docker-compose -f docker-compose.debug.yml logs -f app-debug

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