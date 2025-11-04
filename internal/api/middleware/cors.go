package middleware

import (
	"net/http"

	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware provides CORS configuration
type CORSMiddleware struct {
	logger *logger.Logger
	config CORSConfig
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(logger *logger.Logger) *CORSMiddleware {
	// Default configuration for development
	config := CORSConfig{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:8080",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
			"http://127.0.0.1:8080",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"Accept",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"X-Total-Count",
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}

	return &CORSMiddleware{
		logger: logger,
		config: config,
	}
}

// Handler returns the CORS middleware handler
func (c *CORSMiddleware) Handler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")

		// Check if origin is allowed
		if origin != "" && c.isOriginAllowed(origin) {
			ctx.Header("Access-Control-Allow-Origin", origin)
		}

		// Set CORS headers
		ctx.Header("Access-Control-Allow-Methods", joinStrings(c.config.AllowMethods, ", "))
		ctx.Header("Access-Control-Allow-Headers", joinStrings(c.config.AllowHeaders, ", "))

		if len(c.config.ExposeHeaders) > 0 {
			ctx.Header("Access-Control-Expose-Headers", joinStrings(c.config.ExposeHeaders, ", "))
		}

		if c.config.AllowCredentials {
			ctx.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.config.MaxAge > 0 {
			ctx.Header("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if ctx.Request.Method == "OPTIONS" {
			c.logger.Debug("CORS preflight request", "origin", origin, "method", ctx.Request.Method)
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}

// isOriginAllowed checks if the origin is in the allowed list
func (c *CORSMiddleware) isOriginAllowed(origin string) bool {
	for _, allowedOrigin := range c.config.AllowOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}
	return false
}

// joinStrings joins a slice of strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}

	return result
}

// SetAllowOrigins updates the allowed origins
func (c *CORSMiddleware) SetAllowOrigins(origins []string) {
	c.config.AllowOrigins = origins
}

// SetAllowCredentials updates the allow credentials setting
func (c *CORSMiddleware) SetAllowCredentials(allow bool) {
	c.config.AllowCredentials = allow
}

// ProductionCORSConfig returns a more restrictive CORS config for production
func ProductionCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{
			// Add your production frontend URLs here
			"https://yourdomain.com",
			"https://www.yourdomain.com",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"Accept",
		},
		ExposeHeaders: []string{
			"Content-Length",
		},
		AllowCredentials: true,
		MaxAge:           3600, // 1 hour
	}
}
