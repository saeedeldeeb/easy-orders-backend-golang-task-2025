# Easy Orders Backend - Setup Guide

## Quick Start (For Interviewers)

The fastest way to get the project running:

```bash
# 1. Clone and enter the project
cd easy-orders-backend-golang-task-2025

# 2. Run complete setup
make setup

# 3. Run tests (including concurrency tests)
make test-concurrency
```

That's it! The project will be running at `http://localhost:8080`

---

## Prerequisites

Before starting, ensure you have:

- **Docker** (20.10+) - [Install Docker](https://docs.docker.com/get-docker/)
- **Docker Compose** (2.0+) - Usually comes with Docker Desktop
- **Make** - Available on macOS/Linux by default
- **Git** - For cloning the repository

### Verify Prerequisites

```bash
docker --version          # Should show Docker version 20.10+
docker-compose --version  # Should show version 2.0+
make --version           # Should be available
```

---

## Setup Instructions

### Option 1: Automated Setup (Recommended)

```bash
make setup
```

This command will:
1. Check prerequisites (Docker, Docker Compose)
2. Create `.env` file from `.env.example`
3. Start all services (PostgreSQL, Redis, Application)
4. Wait for services to be healthy
5. Show next steps

### Option 2: Manual Setup

```bash
# 1. Create environment file
cp .env.example .env

# 2. (Optional) Modify .env if needed
# Default values work out of the box

# 3. Start development environment
make dev

# 4. Wait for services to be ready
# Check with: docker-compose -f docker-compose.dev.yml ps

# 5. View logs
make logs-dev
```

---

## Running the Application

### Docker Compose Files Explained

The project has **two Docker Compose configurations**:

#### 1. `docker-compose.yml` - Production/Normal Setup
- **Purpose**: Standard production-like environment
- **Application**: Built from `Dockerfile`, runs compiled binary
- **Volume**: Read-only volume (`./:/app:ro`) - no hot reload
- **Use case**: Testing production behavior, final testing

#### 2. `docker-compose.dev.yml` - Development with Hot Reload
- **Purpose**: Development environment with live code reloading
- **Application**: Uses `Dockerfile.dev` with Air (hot reload tool)
- **Volume**: Read-write volume (`./:/app`) - code changes trigger reload
- **Use case**: Active development, testing changes instantly
- **⚡ Recommended for development**

### Services in Docker Compose

Both configurations include the following services:

```yaml
Services:
├── postgres          # PostgreSQL 15 Database
│   ├── Port: 5433:5432 (external:internal)
│   ├── Database: easy_orders
│   └── Healthcheck: pg_isready
│
├── redis            # Redis 7 (Caching & Sessions)
│   ├── Port: 6379:6379
│   └── Healthcheck: redis-cli ping
│
├── app / app-dev    # Go Application
│   ├── Port: 8080:8080
│   ├── Depends on: postgres, redis
│   ├── Normal: Compiled binary
│   └── Dev: Air hot reload
│
└── adminer          # Database Management UI
    ├── Port: 8081:8080
    └── URL: http://localhost:8081
```

### Development Mode (with Hot Reload) ⚡

**Recommended for active development:**

```bash
# Start development environment
make dev

# Services available:
# - Application: http://localhost:8080 (with hot reload!)
# - Database: localhost:5433 (PostgreSQL)
# - Redis: localhost:6379
# - Adminer UI: http://localhost:8081

# Any code changes automatically reload the app!
# Edit code → Save → Air detects changes → Rebuilds → Restarts
```

**Hot Reload in Action:**
```bash
# 1. Start dev environment
make dev

# 2. Make code changes in your editor
# 3. Save the file
# 4. Check logs to see reload
make logs-dev

# Output shows:
# Air: watching directory...
# Air: building...
# Air: running...
```

### Production Mode

**For testing production-like behavior:**

```bash
# Build and run production containers
make docker-up

# Application runs at: http://localhost:8080
# Uses compiled binary (no hot reload)
```

### Stop Services

```bash
# Stop development environment
make dev-down

# Stop production environment
make docker-down
```

## Seeded Users

The application comes with pre-seeded test users for your convenience (see `migrations/migrations.go:103`):

| Email | Password | Name | Role |
|-------|----------|------|------|
| `admin@easy-orders.com` | `password` | System Administrator | Admin |
| `customer@example.com` | `password` | John Doe | Customer |

These users are automatically created when the database migrations run.

### ⚠️ Important Testing Warning

**Tests run on the development database for minimal setup and ease of use.** This means:

- Running tests may modify or delete data in your dev database
- **After running the concurrency integration tests, the seeded users will be deleted**
- This is intentional to keep the test environment simple and avoid complex test database configuration
- If users are deleted, simply restart the application to re-seed them:
  ```bash
  make dev-down
  make dev
  ```

---

## Testing

### Run All Tests

```bash
make test
```

### Run Concurrency Tests (KEY FEATURE)

This project implements **concurrent order processing** with proper transaction management:

```bash
make test-concurrency
```

**What this tests:**
- ✅ **No overselling**: 100 concurrent orders for 100 stock → exactly 10 succeed
- ✅ **Transaction atomicity**: Failed orders don't reserve inventory
- ✅ **Row-level locking**: SELECT FOR UPDATE prevents race conditions
- ✅ **Optimistic locking**: Version field prevents lost updates
- ✅ **Rollback mechanism**: Partial failures undo all changes

**⚠️ Warning**: Running this test will delete seeded users from the development database. To restore them, restart the application with `make dev-down && make dev`.

### Run Integration Tests

```bash
make test-integration
```

### Run Unit Tests

```bash
make test-unit
```

---

## Debugging

### Debug with Delve (GoLand/VS Code)

```bash
# Start debug environment
make debug

# Delve debugger listens on localhost:2345
# Attach your IDE debugger to this port
```

## License

This is a technical assessment project.
