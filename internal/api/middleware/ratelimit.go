package middleware

import (
	"net/http"
	"sync"
	"time"

	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RateLimiter represents a rate limiter for API endpoints
type RateLimiter struct {
	requests map[string]*UserRateLimit
	mutex    sync.RWMutex
	logger   *logger.Logger

	// Configuration
	maxRequests     int           // Maximum requests per window
	window          time.Duration // Time window for rate limiting
	cleanupInterval time.Duration // Cleanup interval for old entries
}

// UserRateLimit tracks rate limiting for a specific user/IP
type UserRateLimit struct {
	count       int
	windowStart time.Time
	lastAccess  time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, window time.Duration, logger *logger.Logger) *RateLimiter {
	rl := &RateLimiter{
		requests:        make(map[string]*UserRateLimit),
		maxRequests:     maxRequests,
		window:          window,
		cleanupInterval: window * 2, // Cleanup old entries twice per window
		logger:          logger,
	}

	// Start cleanup goroutine
	go rl.cleanupRoutine()

	return rl
}

// Limit returns a middleware that enforces rate limiting
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get identifier (prefer user ID, fallback to IP)
		identifier := rl.getIdentifier(c)

		// Check rate limit
		if !rl.isAllowed(identifier) {
			rl.logger.Warn("Rate limit exceeded", "identifier", identifier, "path", c.Request.URL.Path)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded. Too many requests.",
				"retry_after": int(rl.window.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// LimitWithCustomConfig returns a middleware with custom rate limiting config
func (rl *RateLimiter) LimitWithCustomConfig(maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := rl.getIdentifier(c)

		if !rl.isAllowedCustom(identifier, maxRequests, window) {
			rl.logger.Warn("Custom rate limit exceeded", "identifier", identifier, "max_requests", maxRequests, "window", window)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded. Too many requests.",
				"retry_after": int(window.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getIdentifier extracts identifier from request (user ID or IP)
func (rl *RateLimiter) getIdentifier(c *gin.Context) string {
	// Try to get user ID from auth context first
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok && userIDStr != "" {
			return "user:" + userIDStr
		}
	}

	// Fallback to IP address
	return "ip:" + c.ClientIP()
}

// isAllowed checks if request is allowed under default rate limit
func (rl *RateLimiter) isAllowed(identifier string) bool {
	return rl.isAllowedCustom(identifier, rl.maxRequests, rl.window)
}

// isAllowedCustom checks if request is allowed under custom rate limit
func (rl *RateLimiter) isAllowedCustom(identifier string, maxRequests int, window time.Duration) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Get or create user rate limit record
	userLimit, exists := rl.requests[identifier]
	if !exists {
		rl.requests[identifier] = &UserRateLimit{
			count:       1,
			windowStart: now,
			lastAccess:  now,
		}
		return true
	}

	// Update last access time
	userLimit.lastAccess = now

	// Check if we're in a new window
	if now.Sub(userLimit.windowStart) >= window {
		// Reset for new window
		userLimit.count = 1
		userLimit.windowStart = now
		return true
	}

	// Check if under limit
	if userLimit.count < maxRequests {
		userLimit.count++
		return true
	}

	// Rate limit exceeded
	return false
}

// cleanupRoutine periodically removes old rate limit entries
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanupOldEntries()
	}
}

// cleanupOldEntries removes old rate limit entries
func (rl *RateLimiter) cleanupOldEntries() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.cleanupInterval)

	for identifier, userLimit := range rl.requests {
		if userLimit.lastAccess.Before(cutoff) {
			delete(rl.requests, identifier)
		}
	}

	rl.logger.Debug("Rate limiter cleanup completed", "remaining_entries", len(rl.requests))
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	return map[string]interface{}{
		"total_tracked_identifiers": len(rl.requests),
		"max_requests_per_window":   rl.maxRequests,
		"window_duration_seconds":   rl.window.Seconds(),
		"cleanup_interval_seconds":  rl.cleanupInterval.Seconds(),
	}
}

// CreateStrictLimiter creates a rate limiter for sensitive endpoints
func CreateStrictLimiter(logger *logger.Logger) *RateLimiter {
	return NewRateLimiter(10, time.Minute, logger) // 10 requests per minute
}

// CreateStandardLimiter creates a rate limiter for normal endpoints
func CreateStandardLimiter(logger *logger.Logger) *RateLimiter {
	return NewRateLimiter(100, time.Minute, logger) // 100 requests per minute
}

// CreateGenerousLimiter creates a rate limiter for high-volume endpoints
func CreateGenerousLimiter(logger *logger.Logger) *RateLimiter {
	return NewRateLimiter(1000, time.Minute, logger) // 1000 requests per minute
}
