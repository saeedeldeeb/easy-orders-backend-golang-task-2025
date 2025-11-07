# Easy Orders Backend Makefile

.PHONY: help build run test test-unit test-integration test-concurrency clean docker-up docker-down docker-build dev debug debug-down debug-logs migrate setup

# Default target
help:
	@echo "==================== Easy Orders Backend ===================="
	@echo ""
	@echo "📦 Setup Commands:"
	@echo "  setup            - Complete project setup (env + dependencies)"
	@echo "  env-setup        - Set up environment configuration from template"
	@echo "  install-tools    - Install development tools"
	@echo ""
	@echo "🐳 Docker Commands:"
	@echo "  dev              - Start development environment with hot reload"
	@echo "  dev-down         - Stop development environment"
	@echo "  docker-up        - Start production services with Docker Compose"
	@echo "  docker-down      - Stop all services"
	@echo "  docker-build     - Build Docker image"
	@echo ""
	@echo "🧪 Testing Commands:"
	@echo "  test             - Run all tests with coverage"
	@echo "  test-unit        - Run unit tests only"
	@echo "  test-integration - Run integration tests (requires DB)"
	@echo "  test-concurrency - Run concurrency tests specifically"
	@echo ""
	@echo "🔨 Build & Run:"
	@echo "  build            - Build the application"
	@echo "  run              - Run the application locally"
	@echo "  clean            - Clean build artifacts"
	@echo ""
	@echo "🐛 Debug Commands:"
	@echo "  debug            - Start debug environment with Delve for GoLand"
	@echo "  debug-down       - Stop debug environment"
	@echo "  debug-logs       - View debug container logs"
	@echo ""
	@echo "🔍 Quality & Utilities:"
	@echo "  lint             - Run linters"
	@echo "  fmt              - Format code"
	@echo "  validate         - Run full validation (env + compile)"
	@echo "  db-shell         - Access PostgreSQL shell"
	@echo "  logs-dev         - View development logs"
	@echo ""
	@echo "============================================================"

# Build the application
build:
	go build -o bin/server cmd/server/main.go

# Run the application locally
run:
	go run cmd/server/main.go

# Complete setup (run this first!)
setup: env-setup
	@echo ""
	@echo "🚀 Setting up Easy Orders Backend..."
	@echo ""
	@echo "1️⃣  Checking prerequisites..."
	@command -v docker >/dev/null 2>&1 || { echo "❌ Docker is not installed. Please install Docker first."; exit 1; }
	@command -v docker-compose >/dev/null 2>&1 || { echo "❌ Docker Compose is not installed. Please install Docker Compose first."; exit 1; }
	@echo "✅ Docker and Docker Compose are installed"
	@echo ""
	@echo "2️⃣  Starting development environment..."
	@make dev
	@echo ""
	@echo "3️⃣  Waiting for services to be ready..."
	@sleep 10
	@echo ""
	@echo "✅ Setup complete!"
	@echo ""
	@echo "📝 Next steps:"
	@echo "   - Run 'make test' to verify everything works"
	@echo "   - Run 'make test-concurrency' to test concurrent order processing"
	@echo "   - View logs with 'make logs-dev'"
	@echo "   - Access database shell with 'make db-shell'"
	@echo ""

# Run all tests with coverage in dev container
test:
	@echo "Running all tests with coverage in dev container..."
	@docker-compose -f docker-compose.dev.yml exec app-dev sh -c "go test -v -coverprofile=coverage.out -covermode=atomic ./tests/... && echo '\n📊 Coverage Summary:' && go tool cover -func=coverage.out | grep total"

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@docker-compose -f docker-compose.dev.yml exec app-dev go test -v ./tests/services/...

# Run integration tests (requires database)
test-integration:
	@echo "Running integration tests..."
	@docker-compose -f docker-compose.dev.yml exec app-dev go test -v -count=1 ./tests/integration/...

# Run concurrency tests specifically
test-concurrency:
	@echo "🔄 Running concurrency tests..."
	@echo "This tests concurrent order processing with SELECT FOR UPDATE, transactions, and optimistic locking"
	@echo ""
	@docker-compose -f docker-compose.dev.yml exec app-dev go test -v -count=1 ./tests/integration/... -run TestOrderConcurrency
	@echo ""
	@echo "✅ Concurrency tests complete!"
	@echo "📊 Tests verify:"
	@echo "   - No overselling with concurrent orders"
	@echo "   - Transaction rollback on failures"
	@echo "   - Row-level locking (SELECT FOR UPDATE)"
	@echo "   - Optimistic locking with version field"

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