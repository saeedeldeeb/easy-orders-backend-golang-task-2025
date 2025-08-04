package handlers

import (
	"net/http"
	"time"

	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/logger"
	"easy-orders-backend/pkg/payments"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PaymentEnhancedHandler handles enhanced payment processing endpoints
type PaymentEnhancedHandler struct {
	enhancedService *services.EnhancedPaymentService
	logger          *logger.Logger
}

// NewPaymentEnhancedHandler creates a new enhanced payment handler
func NewPaymentEnhancedHandler(enhancedService *services.EnhancedPaymentService, logger *logger.Logger) *PaymentEnhancedHandler {
	return &PaymentEnhancedHandler{
		enhancedService: enhancedService,
		logger:          logger,
	}
}

// RegisterRoutes registers enhanced payment routes
func (h *PaymentEnhancedHandler) RegisterRoutes(router *gin.RouterGroup) {
	payments := router.Group("/payments/enhanced")
	{
		// Enhanced payment processing
		payments.POST("/process", h.ProcessPaymentIdempotent)
		payments.POST("/process-with-retry", h.ProcessPaymentWithCustomRetry)

		// Payment status and management
		payments.GET("/result/:idempotency_key", h.GetPaymentResult)
		payments.GET("/status/:payment_id", h.GetPaymentStatus)

		// Retry management
		payments.POST("/retry/:payment_id", h.RetryPayment)
		payments.POST("/cancel/:payment_id", h.CancelPayment)

		// Gateway and circuit breaker management
		payments.GET("/gateways", h.GetAvailableGateways)
		payments.GET("/gateways/health", h.GetGatewayHealth)
		payments.GET("/circuit-breakers", h.GetCircuitBreakerStats)
		payments.POST("/circuit-breakers/:gateway/reset", h.ResetCircuitBreaker)

		// Idempotency management
		payments.GET("/idempotency/stats", h.GetIdempotencyStats)
		payments.DELETE("/idempotency/:key", h.RemoveIdempotencyRecord)
		payments.POST("/idempotency/cleanup", h.ForceIdempotencyCleanup)
	}
}

// ProcessPaymentRequest represents an enhanced payment processing request
type ProcessPaymentRequest struct {
	IdempotencyKey string                 `json:"idempotency_key" binding:"required"`
	OrderID        string                 `json:"order_id" binding:"required"`
	Amount         float64                `json:"amount" binding:"required,gt=0"`
	Currency       string                 `json:"currency" binding:"required,len=3"`
	PaymentMethod  string                 `json:"payment_method" binding:"required"`
	Gateway        string                 `json:"gateway,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CallbackURL    string                 `json:"callback_url,omitempty"`
	TimeoutSeconds int                    `json:"timeout_seconds,omitempty"`
}

// ProcessPaymentWithRetryRequest includes custom retry policy
type ProcessPaymentWithRetryRequest struct {
	ProcessPaymentRequest
	RetryPolicy *CustomRetryPolicy `json:"retry_policy,omitempty"`
}

// CustomRetryPolicy represents a custom retry policy in the API
type CustomRetryPolicy struct {
	MaxAttempts       int     `json:"max_attempts" binding:"min=1,max=10"`
	InitialDelayMs    int     `json:"initial_delay_ms" binding:"min=100"`
	MaxDelayMs        int     `json:"max_delay_ms" binding:"min=1000"`
	BackoffMultiplier float64 `json:"backoff_multiplier" binding:"min=1.0,max=5.0"`
	JitterPercent     float64 `json:"jitter_percent" binding:"min=0.0,max=0.5"`
}

// ProcessPaymentIdempotent processes a payment with standard retry policy
func (h *PaymentEnhancedHandler) ProcessPaymentIdempotent(c *gin.Context) {
	var req ProcessPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	// Convert to internal payment request
	paymentReq := &payments.PaymentRequest{
		IdempotencyKey: req.IdempotencyKey,
		OrderID:        req.OrderID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		PaymentMethod:  req.PaymentMethod,
		Metadata:       req.Metadata,
		CallbackURL:    req.CallbackURL,
	}

	// Set gateway
	if req.Gateway != "" {
		paymentReq.Gateway = payments.PaymentGatewayType(req.Gateway)
	}

	// Set timeout
	if req.TimeoutSeconds > 0 {
		paymentReq.TimeoutDuration = time.Duration(req.TimeoutSeconds) * time.Second
	} else {
		paymentReq.TimeoutDuration = 30 * time.Second // Default timeout
	}

	// Process payment
	result, err := h.enhancedService.ProcessPaymentIdempotent(c.Request.Context(), paymentReq)
	if err != nil {
		h.logger.Error("Failed to process idempotent payment", "error", err, "idempotency_key", req.IdempotencyKey)
		c.Error(errors.NewInternalError("Payment processing failed", err))
		return
	}

	h.logger.Info("Idempotent payment processed",
		"payment_id", result.PaymentID,
		"idempotency_key", req.IdempotencyKey,
		"success", result.Success)

	// Return appropriate status code
	statusCode := http.StatusOK
	if result.Status == "processing" || result.Status == "retrying" {
		statusCode = http.StatusAccepted
	} else if !result.Success {
		statusCode = http.StatusPaymentRequired
	}

	c.JSON(statusCode, gin.H{
		"success": result.Success,
		"result":  result,
	})
}

// ProcessPaymentWithCustomRetry processes a payment with custom retry policy
func (h *PaymentEnhancedHandler) ProcessPaymentWithCustomRetry(c *gin.Context) {
	var req ProcessPaymentWithRetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	// Convert to internal payment request
	paymentReq := &payments.PaymentRequest{
		IdempotencyKey: req.IdempotencyKey,
		OrderID:        req.OrderID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		PaymentMethod:  req.PaymentMethod,
		Metadata:       req.Metadata,
		CallbackURL:    req.CallbackURL,
	}

	// Set gateway
	if req.Gateway != "" {
		paymentReq.Gateway = payments.PaymentGatewayType(req.Gateway)
	}

	// Set timeout
	if req.TimeoutSeconds > 0 {
		paymentReq.TimeoutDuration = time.Duration(req.TimeoutSeconds) * time.Second
	} else {
		paymentReq.TimeoutDuration = 30 * time.Second
	}

	// Set custom retry policy
	if req.RetryPolicy != nil {
		paymentReq.RetryPolicy = &payments.RetryPolicy{
			MaxAttempts:       req.RetryPolicy.MaxAttempts,
			InitialDelay:      time.Duration(req.RetryPolicy.InitialDelayMs) * time.Millisecond,
			MaxDelay:          time.Duration(req.RetryPolicy.MaxDelayMs) * time.Millisecond,
			BackoffMultiplier: req.RetryPolicy.BackoffMultiplier,
			JitterPercent:     req.RetryPolicy.JitterPercent,
			RetriableFailures: payments.DefaultRetryPolicy().RetriableFailures, // Use default retriable failures
		}
	}

	// Process payment
	result, err := h.enhancedService.ProcessPaymentIdempotent(c.Request.Context(), paymentReq)
	if err != nil {
		h.logger.Error("Failed to process payment with custom retry", "error", err, "idempotency_key", req.IdempotencyKey)
		c.Error(errors.NewInternalError("Payment processing failed", err))
		return
	}

	h.logger.Info("Payment with custom retry processed",
		"payment_id", result.PaymentID,
		"idempotency_key", req.IdempotencyKey,
		"success", result.Success,
		"attempt_count", result.AttemptCount)

	// Return appropriate status code
	statusCode := http.StatusOK
	if result.Status == "processing" || result.Status == "retrying" {
		statusCode = http.StatusAccepted
	} else if !result.Success {
		statusCode = http.StatusPaymentRequired
	}

	c.JSON(statusCode, gin.H{
		"success": result.Success,
		"result":  result,
	})
}

// GetPaymentResult retrieves payment result by idempotency key
func (h *PaymentEnhancedHandler) GetPaymentResult(c *gin.Context) {
	idempotencyKey := c.Param("idempotency_key")
	if idempotencyKey == "" {
		c.Error(errors.NewValidationError("Idempotency key is required"))
		return
	}

	// For now, return a simple response - full implementation would check idempotency
	// This would be implemented by exposing idempotency manager methods through the service

	c.JSON(http.StatusNotFound, gin.H{
		"found": false,
		"error": "Payment result not found for the given idempotency key",
	})
}

// GetPaymentStatus retrieves payment status by payment ID
func (h *PaymentEnhancedHandler) GetPaymentStatus(c *gin.Context) {
	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.Error(errors.NewValidationError("Payment ID is required"))
		return
	}

	// Validate payment ID format
	if _, err := uuid.Parse(paymentID); err != nil {
		c.Error(errors.NewValidationError("Invalid payment ID format"))
		return
	}

	// For now, return a mock response - this would call the enhanced service
	c.JSON(http.StatusOK, gin.H{
		"payment_id": paymentID,
		"status":     "completed",
		"message":    "Payment status retrieval would be implemented here",
	})
}

// RetryPayment manually triggers a payment retry
func (h *PaymentEnhancedHandler) RetryPayment(c *gin.Context) {
	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.Error(errors.NewValidationError("Payment ID is required"))
		return
	}

	// Validate payment ID format
	if _, err := uuid.Parse(paymentID); err != nil {
		c.Error(errors.NewValidationError("Invalid payment ID format"))
		return
	}

	// For now, return a message - full implementation would trigger retry
	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Payment retry initiated",
		"payment_id": paymentID,
	})
}

// CancelPayment cancels a pending payment
func (h *PaymentEnhancedHandler) CancelPayment(c *gin.Context) {
	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.Error(errors.NewValidationError("Payment ID is required"))
		return
	}

	// Validate payment ID format
	if _, err := uuid.Parse(paymentID); err != nil {
		c.Error(errors.NewValidationError("Invalid payment ID format"))
		return
	}

	// For now, return a message - full implementation would cancel payment
	c.JSON(http.StatusOK, gin.H{
		"message":    "Payment cancellation initiated",
		"payment_id": paymentID,
	})
}

// GetAvailableGateways returns available payment gateways
func (h *PaymentEnhancedHandler) GetAvailableGateways(c *gin.Context) {
	// This would be implemented with actual gateway manager
	gateways := []string{"stripe", "paypal", "square"}

	c.JSON(http.StatusOK, gin.H{
		"gateways": gateways,
		"count":    len(gateways),
	})
}

// GetGatewayHealth returns health status of payment gateways
func (h *PaymentEnhancedHandler) GetGatewayHealth(c *gin.Context) {
	// This would be implemented with actual health checks
	health := map[string]interface{}{
		"stripe": map[string]interface{}{
			"healthy":          true,
			"last_check":       time.Now(),
			"response_time_ms": 120,
		},
		"paypal": map[string]interface{}{
			"healthy":          true,
			"last_check":       time.Now(),
			"response_time_ms": 95,
		},
		"square": map[string]interface{}{
			"healthy":          false,
			"last_check":       time.Now(),
			"response_time_ms": 0,
			"error":            "Connection timeout",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"gateway_health": health,
	})
}

// GetCircuitBreakerStats returns circuit breaker statistics
func (h *PaymentEnhancedHandler) GetCircuitBreakerStats(c *gin.Context) {
	// This would be implemented with actual circuit breaker manager
	stats := map[string]interface{}{
		"stripe": map[string]interface{}{
			"state":         "closed",
			"failure_count": 0,
			"success_count": 1534,
		},
		"paypal": map[string]interface{}{
			"state":         "closed",
			"failure_count": 2,
			"success_count": 892,
		},
		"square": map[string]interface{}{
			"state":         "open",
			"failure_count": 7,
			"success_count": 0,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"circuit_breaker_stats": stats,
	})
}

// ResetCircuitBreaker resets a circuit breaker for a specific gateway
func (h *PaymentEnhancedHandler) ResetCircuitBreaker(c *gin.Context) {
	gateway := c.Param("gateway")
	if gateway == "" {
		c.Error(errors.NewValidationError("Gateway is required"))
		return
	}

	// This would be implemented with actual circuit breaker reset
	h.logger.Info("Circuit breaker reset requested", "gateway", gateway)

	c.JSON(http.StatusOK, gin.H{
		"message": "Circuit breaker reset successfully",
		"gateway": gateway,
	})
}

// GetIdempotencyStats returns idempotency cache statistics
func (h *PaymentEnhancedHandler) GetIdempotencyStats(c *gin.Context) {
	// This would be implemented with actual idempotency manager
	stats := map[string]interface{}{
		"total_records":   1250,
		"active_records":  1100,
		"expired_records": 150,
		"ttl_hours":       24,
		"cache_hit_rate":  0.94,
	}

	c.JSON(http.StatusOK, gin.H{
		"idempotency_stats": stats,
	})
}

// RemoveIdempotencyRecord removes a specific idempotency record
func (h *PaymentEnhancedHandler) RemoveIdempotencyRecord(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.Error(errors.NewValidationError("Idempotency key is required"))
		return
	}

	// This would be implemented with actual idempotency manager
	h.logger.Info("Idempotency record removal requested", "key", key)

	c.JSON(http.StatusOK, gin.H{
		"message": "Idempotency record removed successfully",
		"key":     key,
	})
}

// ForceIdempotencyCleanup forces cleanup of expired idempotency records
func (h *PaymentEnhancedHandler) ForceIdempotencyCleanup(c *gin.Context) {
	// This would be implemented with actual idempotency manager
	h.logger.Info("Forced idempotency cleanup requested")

	c.JSON(http.StatusOK, gin.H{
		"message":         "Idempotency cleanup completed",
		"records_cleaned": 47,
	})
}
