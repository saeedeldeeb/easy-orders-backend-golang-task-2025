package middleware

import (
	"net/http"
	"strings"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/jwt"
	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	tokenManager *jwt.TokenManager
	logger       *logger.Logger
}

// NewAuthMiddleware creates new auth middleware
func NewAuthMiddleware(tokenManager *jwt.TokenManager, logger *logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		tokenManager: tokenManager,
		logger:       logger,
	}
}

// RequireAuth is a middleware that requires valid JWT authentication
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			m.logger.Warn("Missing Authorization header", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check Bearer prefix
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			m.logger.Warn("Invalid Authorization header format", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header must be in format 'Bearer <token>'",
			})
			c.Abort()
			return
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, bearerPrefix)
		if token == "" {
			m.logger.Warn("Empty token in Authorization header", "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token is required",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := m.tokenManager.ValidateToken(token)
		if err != nil {
			m.logger.Warn("Invalid token", "error", err, "path", c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("claims", claims)

		m.logger.Debug("User authenticated", "user_id", claims.UserID, "role", claims.Role, "path", c.Request.URL.Path)

		c.Next()
	}
}

// RequireRole is a middleware that requires specific user roles
func (m *AuthMiddleware) RequireRole(allowedRoles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user role from context (set by RequireAuth middleware)
		userRole, exists := c.Get("user_role")
		if !exists {
			m.logger.Error("User role not found in context", "path", c.Request.URL.Path)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Authentication context not found",
			})
			c.Abort()
			return
		}

		role, ok := userRole.(models.UserRole)
		if !ok {
			m.logger.Error("Invalid user role type in context", "role", userRole, "path", c.Request.URL.Path)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid authentication context",
			})
			c.Abort()
			return
		}

		// Check if user role is allowed
		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				m.logger.Debug("User role authorized", "user_role", role, "path", c.Request.URL.Path)
				c.Next()
				return
			}
		}

		// User doesn't have required role
		userID, _ := c.Get("user_id")
		m.logger.Warn("User access denied", "user_id", userID, "user_role", role, "required_roles", allowedRoles, "path", c.Request.URL.Path)
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions",
		})
		c.Abort()
	}
}

// RequireAdmin is a convenience middleware that requires admin role
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return m.RequireRole(models.UserRoleAdmin)
}

// RequireCustomerOrAdmin allows both customer and admin roles
func (m *AuthMiddleware) RequireCustomerOrAdmin() gin.HandlerFunc {
	return m.RequireRole(models.UserRoleCustomer, models.UserRoleAdmin)
}

// OptionalAuth is a middleware that extracts user info if token is present but doesn't require it
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header, continue without user context
			c.Next()
			return
		}

		// Check Bearer prefix
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			// Invalid format, continue without user context
			c.Next()
			return
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, bearerPrefix)
		if token == "" {
			// Empty token, continue without user context
			c.Next()
			return
		}

		// Validate token
		claims, err := m.tokenManager.ValidateToken(token)
		if err != nil {
			// Invalid token, continue without user context
			m.logger.Debug("Optional auth failed", "error", err, "path", c.Request.URL.Path)
			c.Next()
			return
		}

		// Store user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("claims", claims)

		m.logger.Debug("Optional auth succeeded", "user_id", claims.UserID, "path", c.Request.URL.Path)

		c.Next()
	}
}

// GetCurrentUser is a helper function to get current user from context
func GetCurrentUser(c *gin.Context) (userID string, userRole models.UserRole, exists bool) {
	userIDValue, userIDExists := c.Get("user_id")
	userRoleValue, userRoleExists := c.Get("user_role")

	if !userIDExists || !userRoleExists {
		return "", "", false
	}

	userID, userIDOk := userIDValue.(string)
	userRole, userRoleOk := userRoleValue.(models.UserRole)

	if !userIDOk || !userRoleOk {
		return "", "", false
	}

	return userID, userRole, true
}

// GetCurrentUserID is a helper function to get current user ID from context
func GetCurrentUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}

	userIDStr, ok := userID.(string)
	return userIDStr, ok
}

// IsCurrentUserAdmin checks if current user is admin
func IsCurrentUserAdmin(c *gin.Context) bool {
	_, role, exists := GetCurrentUser(c)
	return exists && role == models.UserRoleAdmin
}
