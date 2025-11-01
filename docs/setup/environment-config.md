# Environment Configuration Guide

## ✅ **Docker Compose with .env File Setup Complete**

The project now uses a **`.env` file** for all Docker Compose environment variables, following best practices for configuration management.

## 📋 **How It Works**

Docker Compose automatically reads the `.env` file and substitutes variables in `docker-compose.yml` and `docker-compose.dev.yml`.

### **Before (Hardcoded):**

```yaml
environment:
  - DB_USER=postgres
  - DB_PASSWORD=postgres
```

### **After (Environment Variables):**

```yaml
environment:
  - DB_USER=${DB_USER}
  - DB_PASSWORD=${DB_PASSWORD}
```

## 🔧 **Environment Variables**

### **Database Configuration**

- `DB_USER` - Database username (default: postgres)
- `DB_PASSWORD` - Database password (default: postgres)
- `DB_NAME` - Database name (default: easy_orders)
- `DB_PORT` - Internal database port (default: 5432)
- `DB_EXTERNAL_PORT` - Host port mapping (default: 5433)
- `DB_SSLMODE` - SSL mode (default: disable)

### **Application Configuration**

- `SERVER_PORT` - API server port (default: 8080)
- `LOG_LEVEL` - Logging level (default: info)
- `ENVIRONMENT` - Environment name (default: docker)
- `JWT_SECRET` - JWT signing secret (CHANGE IN PRODUCTION)

### **Development Overrides**

- `JWT_SECRET_DEV` - Development JWT secret
- `LOG_LEVEL_DEV` - Development logging (default: debug)
- `ENVIRONMENT_DEV` - Development environment name

### **Redis Configuration**

- `REDIS_HOST` - Redis hostname (default: localhost)
- `REDIS_PORT` - Redis port (default: 6379)
- `REDIS_PASSWORD` - Redis password (optional)

### **Docker Services**

- `ADMINER_PORT` - Database admin UI port (default: 8081)

## 🚀 **Usage**

### **1. Copy Environment Template**

```bash
cp .env.example .env
```

### **2. Edit Configuration**

```bash
# Edit .env file with your preferred settings
vim .env
```

### **3. Start Services**

```bash
# Production configuration
docker-compose up -d

# Development configuration (uses *_DEV variables)
docker-compose -f docker-compose.dev.yml up -d
```

## 🔒 **Security Best Practices**

### **Production Checklist:**

- [ ] Change `JWT_SECRET` to a strong random string
- [ ] Use strong `DB_PASSWORD`
- [ ] Set appropriate `LOG_LEVEL` (warn or error)
- [ ] Review all default values

### **Example Production .env:**

```bash
# Production Database
DB_PASSWORD=your-super-secure-database-password-here
JWT_SECRET=your-super-secure-jwt-secret-256-bits-long

# Production Logging
LOG_LEVEL=warn
ENVIRONMENT=production
```

## 📁 **File Structure**

```
/
├── .env                 # Your local configuration (not committed)
├── .env.example         # Template for new users (committed)
├── docker-compose.yml   # Production setup (uses .env)
└── docker-compose.dev.yml # Development setup (uses .env)
```

## 🎯 **Connection Details**

With the default `.env` configuration:

### **Database:**

- **Host:** localhost:5433
- **Username:** postgres
- **Password:** postgres
- **Database:** easy_orders

### **Services:**

- **API Server:** http://localhost:8080
- **Redis:** localhost:6379
- **Adminer:** http://localhost:8081

## ✨ **Benefits**

1. **Flexibility** - Easy to change ports and configurations
2. **Security** - Sensitive values in `.env` (not committed to git)
3. **Environment-specific** - Different configs for dev/prod
4. **Clean** - No hardcoded values in docker-compose files
5. **Standard** - Follows Docker Compose best practices

## 🔄 **Updating Configuration**

To change any setting:

1. Edit `.env` file
2. Restart affected services:
   ```bash
   docker-compose down
   docker-compose up -d
   ```

The setup is now **production-ready** and follows Docker best practices! 🚀
