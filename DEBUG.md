# Debugging Guide for Easy Orders Backend

This guide explains how to debug the Easy Orders Go backend application running in Docker containers using GoLand.

## Table of Contents
- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [GoLand Setup](#goland-setup)
- [Debugging Workflow](#debugging-workflow)
- [Switching Between Dev and Debug Modes](#switching-between-dev-and-debug-modes)
- [Troubleshooting](#troubleshooting)
- [Technical Details](#technical-details)

## Overview

The project provides two separate containerized environments:

1. **Development Mode** (`docker-compose.dev.yml`):
   - Uses Air for hot-reload functionality
   - Automatically rebuilds and restarts on code changes
   - Fast iteration for rapid development
   - Command: `make dev`

2. **Debug Mode** (`docker-compose.debug.yml`):
   - Uses Delve debugger
   - Allows setting breakpoints and step-by-step debugging
   - Full GoLand integration
   - Command: `make debug`

**Important**: Air and Delve cannot run simultaneously because Air constantly restarts the application, which would disconnect your debug sessions. Use dev mode for coding, and debug mode when you need to investigate issues.

## Prerequisites

- **GoLand IDE** (or IntelliJ IDEA Ultimate with Go plugin)
- **Docker** and **Docker Compose** installed
- **Make** command-line tool
- Project `.env` file configured (see `env-setup` in main README)

## Quick Start

### 1. Start Debug Environment

```bash
make debug
```

This will:
- Build the debug Docker image with Delve
- Start all services (app-debug, postgres, redis, adminer)
- Expose Delve debugger on port 2345
- Wait for services to be ready

### 2. Attach Debugger in GoLand

1. Open the project in GoLand
2. Set breakpoints in your code by clicking in the gutter next to line numbers
3. Select the **"Debug in Docker"** run configuration from the dropdown in the toolbar
4. Click the **Debug** button (bug icon) or press **Shift+F9**

GoLand will automatically:
- Connect to Delve on `localhost:2345`
- Map your local source code to the container's `/app` directory
- Stop at your breakpoints when they're hit

### 3. Debug Your Code

Once attached:
- **F8**: Step over
- **F7**: Step into
- **Shift+F8**: Step out
- **Alt+F9**: Run to cursor
- **F9**: Resume program
- **Ctrl+F8**: Toggle breakpoint

### 4. Stop Debug Environment

When finished debugging:

```bash
make debug-down
```

## GoLand Setup

### Automatic Setup (Recommended)

The project includes pre-configured run configurations in the `.run/` directory:

- **"Debug in Docker"**: Remote debug configuration for attaching to Delve
- **"Start Debug Container"**: Shell script to start the debug container

GoLand should automatically detect these configurations when you open the project.

### Manual Setup (if needed)

If the configurations don't appear:

1. **Create Remote Debug Configuration**:
   - Go to **Run** → **Edit Configurations**
   - Click **+** → **Go Remote**
   - Name: `Debug in Docker`
   - Host: `localhost`
   - Port: `2345`
   - Click **OK**

2. **Add Path Mappings** (if needed):
   - In the same configuration dialog
   - Under **"Path mappings"** section
   - Add mapping: `<Project Root>` → `/app`

### Verifying Setup

1. Check that `.run/Debug in Docker.run.xml` exists
2. Check that `.run/Start Debug Container.run.xml` exists
3. Verify configurations appear in GoLand's run configuration dropdown

## Debugging Workflow

### Typical Debugging Session

```bash
# 1. Stop dev environment if running
make docker-down  # or Ctrl+C if running in foreground

# 2. Start debug environment
make debug

# 3. In GoLand: Set breakpoints in your code

# 4. Start debugging (Shift+F9 or click Debug button)

# 5. Trigger the code path (e.g., make API request)
curl http://localhost:8080/api/v1/your-endpoint

# 6. Debug in GoLand when breakpoint hits

# 7. When done, stop debug environment
make debug-down

# 8. Return to dev mode for continued development
make dev
```

### Viewing Logs

While debugging, you can monitor logs in a separate terminal:

```bash
make debug-logs
```

This shows real-time logs from the debug container, including:
- Application logs
- Delve debugger output
- Database connection status
- HTTP requests

## Switching Between Dev and Debug Modes

### From Dev to Debug

```bash
# Stop development environment
docker-compose -f docker-compose.dev.yml down
# or
make docker-down  # if using production compose

# Start debug environment
make debug
```

### From Debug to Dev

```bash
# Stop debug environment
make debug-down

# Start development environment
make dev
```

### Quick Commands

```bash
# Check what's running
docker ps

# View available make commands
make help

# Stop all Docker containers
docker-compose down && docker-compose -f docker-compose.dev.yml down && docker-compose -f docker-compose.debug.yml down
```

## Troubleshooting

### Debugger Won't Connect

**Problem**: GoLand shows "Failed to connect to remote debugger"

**Solutions**:
1. Verify debug container is running:
   ```bash
   docker ps | grep easy-orders-app-debug
   ```

2. Check Delve is listening:
   ```bash
   make debug-logs
   # Look for: "API server listening at: [::]:2345"
   ```

3. Verify port 2345 is not in use:
   ```bash
   lsof -i :2345
   ```

4. Rebuild debug image:
   ```bash
   make debug-down
   docker-compose -f docker-compose.debug.yml build --no-cache
   make debug
   ```

### Breakpoints Not Hit

**Problem**: Debugger connects but breakpoints are ignored

**Solutions**:
1. Verify path mappings in GoLand:
   - Check **Run** → **Edit Configurations** → **Debug in Docker**
   - Ensure mapping: `<Project Root>` → `/app`

2. Check that source code is mounted:
   ```bash
   docker exec -it easy-orders-app-debug ls -la /app
   ```

3. Ensure code is up-to-date in container:
   ```bash
   make debug-down
   make debug
   ```

### Application Crashes on Startup

**Problem**: Container starts but application crashes immediately

**Solutions**:
1. Check environment variables:
   ```bash
   make debug-logs
   ```

2. Verify `.env` file exists and is valid:
   ```bash
   make env-check
   ```

3. Check database connectivity:
   ```bash
   docker-compose -f docker-compose.debug.yml exec postgres pg_isready
   ```

4. Review application logs for stack traces:
   ```bash
   make debug-logs
   ```

### Port Conflicts

**Problem**: Port 2345 or 8080 already in use

**Solutions**:
1. Find and stop conflicting process:
   ```bash
   lsof -ti:2345 | xargs kill -9
   lsof -ti:8080 | xargs kill -9
   ```

2. Or change ports in `docker-compose.debug.yml`:
   ```yaml
   ports:
     - '8081:8080'  # Change external port
     - '2346:2345'  # Change Delve port
   ```
   Then update GoLand configuration to use new port.

### Slow Debugging Performance

**Problem**: Stepping through code is slow

**Solutions**:
1. The debug image disables compiler optimizations (`-gcflags "all=-N -l"`) for accurate debugging - this is expected
2. Consider debugging specific sections rather than stepping through everything
3. Use conditional breakpoints to reduce stops
4. For performance testing, use dev or production builds instead

## Technical Details

### Delve Configuration

The debug container runs Delve with these options:
```bash
dlv exec ./main --headless --listen=:2345 --api-version=2 --accept-multiclient --continue
```

- `--headless`: Run without terminal UI
- `--listen=:2345`: Listen on all interfaces, port 2345
- `--api-version=2`: Use API v2 for better IDE integration
- `--accept-multiclient`: Allow multiple debugger connections
- `--continue`: Start application immediately (debugger controls execution)

### Build Flags

Debug builds use special flags:
```bash
go build -gcflags "all=-N -l" -o main cmd/server/main.go
```

- `-gcflags "all=-N -l"`: Disable compiler optimizations
  - `-N`: Disable optimizations
  - `-l`: Disable inlining

These flags make debugging more accurate but reduce performance.

### Container Security

The debug container requires additional privileges:
```yaml
security_opt:
  - "apparmor:unconfined"
cap_add:
  - SYS_PTRACE
```

- `apparmor:unconfined`: Required for ptrace operations
- `SYS_PTRACE`: Allows debugger to inspect process memory

**Warning**: Only use debug mode in development environments, never in production.

### Architecture

```
┌─────────────┐         ┌──────────────────┐
│   GoLand    │────────▶│  Delve Debugger  │
│             │  :2345  │  (in container)  │
└─────────────┘         └──────────────────┘
                                │
                                ▼
                        ┌──────────────────┐
                        │   Go App (main)  │
                        │   Port :8080     │
                        └──────────────────┘
                                │
                        ┌───────┴────────┐
                        ▼                ▼
                 ┌───────────┐    ┌──────────┐
                 │ PostgreSQL│    │  Redis   │
                 │  :5432    │    │  :6379   │
                 └───────────┘    └──────────┘
```

### File Structure

```
.
├── docker/
│   ├── Dockerfile.dev      # Hot-reload development
│   └── Dockerfile.debug    # Delve debugger
├── docker-compose.dev.yml  # Dev environment config
├── docker-compose.debug.yml # Debug environment config
├── .run/
│   ├── Debug in Docker.run.xml       # GoLand debug config
│   └── Start Debug Container.run.xml # Container startup
├── Makefile                # Convenient commands
└── DEBUG.md               # This file
```

## Additional Resources

- [Delve Documentation](https://github.com/go-delve/delve)
- [GoLand Remote Debugging Guide](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html)
- [Docker Security Best Practices](https://docs.docker.com/engine/security/)
- [Go Debugging Tips](https://golang.org/doc/diagnostics)

## Common Use Cases

### Debugging API Endpoints

1. Set breakpoint in handler (e.g., `internal/api/handlers/product_handler.go`)
2. Start debug session
3. Make API request:
   ```bash
   curl -X POST http://localhost:8080/api/v1/products \
     -H "Content-Type: application/json" \
     -d '{"name": "Test Product"}'
   ```
4. GoLand will pause at your breakpoint

### Debugging Database Queries

1. Set breakpoint in repository method (e.g., `internal/repository/product_repository.go`)
2. Start debug session
3. Trigger database operation via API or test
4. Inspect query parameters and results in debugger

### Debugging Middleware

1. Set breakpoint in middleware (e.g., `internal/api/middleware/auth.go`)
2. Start debug session
3. Make authenticated request
4. Step through auth validation logic

### Debugging Background Workers

1. Set breakpoint in worker code (e.g., `pkg/workers/`)
2. Start debug session
3. Worker will hit breakpoint when scheduled task runs
4. Inspect worker state and execution flow

## FAQ

**Q: Can I use both dev and debug environments at the same time?**

A: No, they share the same database and ports. Stop one before starting the other.

**Q: Will my breakpoints work if I modify code while debugging?**

A: No, you need to rebuild. Stop the debugger, let the changes save, then restart debug session.

**Q: Can I debug tests with this setup?**

A: This setup is for debugging the running application. For test debugging, use GoLand's built-in test runner with local debugging (no Docker needed).

**Q: Why is debugging slower than normal execution?**

A: Debug builds disable optimizations for accurate debugging. This is normal and expected.

**Q: Can I use print statements instead of debugging?**

A: Yes, but proper debugging is more efficient. Add logs with `pkg/logger` and view them with `make debug-logs`.

---

**Happy Debugging!** 🐛
