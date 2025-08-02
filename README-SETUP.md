# Easy Orders Backend - Setup Guide

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Make (optional, for convenience commands)

### 1. Clone and Setup

```bash
git clone <repository-url>
cd easy-orders-backend-golang-task-2025
cp .env.example .env
```

### 2. Start with Docker (Recommended)

```bash
# Start all services (PostgreSQL, Redis, App)
make docker-up
# OR
docker-compose up -d

# Check if services are running
docker-compose ps
```

### 3. Development Mode (Hot Reload)

```bash
# Start development environment with hot reload
make dev
# OR
docker-compose -f docker-compose.dev.yml up
```

### 4. Local Development (Without Docker)

```bash
# Install development tools
make install-tools

# Start PostgreSQL and Redis with Docker
docker-compose up postgres redis -d

# Update .env with local settings
# Run the app locally
make run
```

## Services

After running `make docker-up`, the following services will be available:

- **API Server**: http://localhost:8080
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379
- **Adminer** (DB Admin): http://localhost:8081

## Useful Commands

```bash
# Build the application
make build

# Run tests
make test

# Run with race detection
make test-race

# Format code
make fmt

# Run linters
make lint

# View logs
make logs

# Database shell
make db-shell

# Redis shell
make redis-shell

# Stop all services
make docker-down

# Clean build artifacts
make clean
```

## Project Structure

```
/
├── cmd/server/          # Application entry point
├── internal/            # Private application code
│   ├── api/            # API handlers, middleware, routes
│   ├── models/         # GORM models
│   ├── services/       # Business logic
│   ├── repository/     # Data access layer
│   ├── config/         # Configuration
│   └── workers/        # Background workers
├── pkg/                # Public packages
│   ├── database/       # Database utilities
│   ├── logger/         # Logging utilities
│   └── utils/          # Common utilities
├── migrations/         # Database migrations
├── tests/             # Test files
├── docker/            # Docker related files
├── docs/              # API documentation
├── .env.example       # Environment template
├── docker-compose.yml # Production compose
└── docker-compose.dev.yml # Development compose
```

## Next Steps

1. The basic structure is ready
2. Next we'll implement the database models and migrations
3. Then the API handlers and business logic
4. Finally the concurrent processing features

## Environment Variables

Copy `.env.example` to `.env` and adjust the values for your environment:

```bash
cp .env.example .env
```

Key variables:

- `JWT_SECRET`: Change to a secure random string
- `DB_*`: Database connection settings
- `SERVER_PORT`: API server port
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
