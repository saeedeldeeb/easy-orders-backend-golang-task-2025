package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"easy-orders-backend/internal/middleware"
	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/jwt"
	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// OrderPipelineHandler handles order pipeline HTTP requests
type OrderPipelineHandler struct {
	pipelineService services.OrderPipelineService
	logger          *logger.Logger
}

// NewOrderPipelineHandler creates a new order pipeline handler
func NewOrderPipelineHandler(
	pipelineService services.OrderPipelineService,
	logger *logger.Logger,
) *OrderPipelineHandler {
	return &OrderPipelineHandler{
		pipelineService: pipelineService,
		logger:          logger,
	}
}

// RegisterRoutes registers all order pipeline routes
func (h *OrderPipelineHandler) RegisterRoutes(router *gin.RouterGroup) {
	pipeline := router.Group("/pipeline")
	{
		pipeline.POST("/orders", h.ProcessOrder)
		pipeline.POST("/orders/async", h.ProcessOrderAsync)
		pipeline.GET("/orders/:id/status", h.GetOrderPipelineStatus)
	}
}

// ProcessOrder handles synchronous order processing through the pipeline
// @Summary Process order through pipeline synchronously
// @Description Processes an order through the concurrent pipeline (placement → inventory → payment → fulfillment → notification)
// @Tags Order Pipeline
// @Accept json
// @Produce json
// @Param request body services.CreateOrderRequest true "Order creation request"
// @Success 200 {object} services.OrderPipelineResult "Pipeline processing result"
// @Failure 400 {object} errors.ErrorResponse "Bad request"
// @Failure 401 {object} errors.ErrorResponse "Unauthorized"
// @Failure 422 {object} errors.ErrorResponse "Business logic error"
// @Failure 500 {object} errors.ErrorResponse "Internal server error"
// @Router /api/v1/pipeline/orders [post]
// @Security BearerAuth
func (h *OrderPipelineHandler) ProcessOrder(c *gin.Context) {
	var req services.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind order pipeline request", "error", err)
		appErr := errors.NewValidationErrorWithDetails("Invalid request body", err.Error())
		middleware.AbortWithError(c, appErr)
		return
	}

	// Extract user ID from JWT claims
	claims, err := extractClaims(c)
	if err != nil {
		h.logger.Error("Failed to extract claims for order pipeline", "error", err)
		appErr := errors.NewUnauthorizedError("Invalid authentication token")
		middleware.AbortWithError(c, appErr)
		return
	}

	// Set user ID from claims if not provided
	if req.UserID == "" {
		req.UserID = claims.UserID
	}

	// Verify user can only create orders for themselves (unless admin)
	if claims.UserID != req.UserID && claims.Role != "admin" {
		h.logger.Warn("User attempted to create order for another user",
			"requesting_user", claims.UserID,
			"order_user", req.UserID)
		appErr := errors.NewForbiddenError("Cannot create orders for other users")
		middleware.AbortWithError(c, appErr)
		return
	}

	// Create a timeout context for the pipeline
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	h.logger.Info("Processing order through pipeline", "user_id", req.UserID, "items_count", len(req.Items))

	// Process order through pipeline
	result, err := h.pipelineService.ProcessOrder(ctx, req)
	if err != nil {
		h.logger.Error("Pipeline processing failed", "error", err, "user_id", req.UserID)
		appErr := errors.NewInternalError("Failed to process order through pipeline", err)
		middleware.AbortWithError(c, appErr)
		return
	}

	// Check if pipeline completed successfully
	if result.Status == services.PipelineStatusFailed {
		h.logger.Error("Order pipeline processing failed",
			"user_id", req.UserID,
			"errors", result.Errors,
			"processing_time", result.ProcessingTime)

		// Return business logic error with pipeline details
		appErr := errors.NewBusinessError("Order processing failed")
		appErr.Context = map[string]interface{}{
			"pipeline_errors": result.Errors,
			"processing_time": result.ProcessingTime.String(),
			"status":          result.Status,
		}
		middleware.AbortWithError(c, appErr)
		return
	}

	h.logger.Info("Order pipeline processing completed",
		"user_id", req.UserID,
		"order_id", getOrderIDFromResult(result),
		"status", result.Status,
		"processing_time", result.ProcessingTime)

	c.JSON(http.StatusOK, result)
}

// ProcessOrderAsync handles asynchronous order processing through the pipeline
// @Summary Process order through pipeline asynchronously
// @Description Starts processing an order through the concurrent pipeline and returns immediately with a channel for results
// @Tags Order Pipeline
// @Accept json
// @Produce json
// @Param request body services.CreateOrderRequest true "Order creation request"
// @Success 202 {object} map[string]interface{} "Async processing started"
// @Failure 400 {object} errors.ErrorResponse "Bad request"
// @Failure 401 {object} errors.ErrorResponse "Unauthorized"
// @Failure 500 {object} errors.ErrorResponse "Internal server error"
// @Router /api/v1/pipeline/orders/async [post]
// @Security BearerAuth
func (h *OrderPipelineHandler) ProcessOrderAsync(c *gin.Context) {
	var req services.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind async order pipeline request", "error", err)
		appErr := errors.NewValidationErrorWithDetails("Invalid request body", err.Error())
		middleware.AbortWithError(c, appErr)
		return
	}

	// Extract user ID from JWT claims
	claims, err := extractClaims(c)
	if err != nil {
		h.logger.Error("Failed to extract claims for async order pipeline", "error", err)
		appErr := errors.NewUnauthorizedError("Invalid authentication token")
		middleware.AbortWithError(c, appErr)
		return
	}

	// Set user ID from claims if not provided
	if req.UserID == "" {
		req.UserID = claims.UserID
	}

	// Verify user can only create orders for themselves (unless admin)
	if claims.UserID != req.UserID && claims.Role != "admin" {
		h.logger.Warn("User attempted to create async order for another user",
			"requesting_user", claims.UserID,
			"order_user", req.UserID)
		appErr := errors.NewForbiddenError("Cannot create orders for other users")
		middleware.AbortWithError(c, appErr)
		return
	}

	h.logger.Info("Starting async order pipeline processing", "user_id", req.UserID, "items_count", len(req.Items))

	// Start async processing
	resultChan, err := h.pipelineService.ProcessOrderAsync(context.Background(), req)
	if err != nil {
		h.logger.Error("Failed to start async pipeline processing", "error", err, "user_id", req.UserID)
		appErr := errors.NewInternalError("Failed to start async order processing", err)
		middleware.AbortWithError(c, appErr)
		return
	}

	// Generate a processing ID for tracking
	processingID := "proc_" + strconv.FormatInt(time.Now().UnixNano(), 36)

	// Start a goroutine to handle the result (in a real implementation, you'd store this in Redis/DB)
	go func() {
		select {
		case result := <-resultChan:
			if result != nil {
				h.logger.Info("Async order pipeline completed",
					"processing_id", processingID,
					"user_id", req.UserID,
					"order_id", getOrderIDFromResult(result),
					"status", result.Status,
					"processing_time", result.ProcessingTime)

				// TODO: Store result in Redis/DB for later retrieval
				// TODO: Send real-time notification via WebSocket if implemented
			}
		case <-time.After(2 * time.Minute):
			h.logger.Warn("Async order pipeline timed out",
				"processing_id", processingID,
				"user_id", req.UserID)
		}
	}()

	h.logger.Info("Async order pipeline started",
		"processing_id", processingID,
		"user_id", req.UserID)

	c.JSON(http.StatusAccepted, gin.H{
		"message":        "Order processing started asynchronously",
		"processing_id":  processingID,
		"user_id":        req.UserID,
		"estimated_time": "30-60 seconds",
		"status":         "processing",
	})
}

// GetOrderPipelineStatus retrieves the status of an order processing pipeline
// @Summary Get order pipeline processing status
// @Description Retrieves the current status and results of an order processing pipeline
// @Tags Order Pipeline
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} services.OrderPipelineResult "Pipeline status and result"
// @Failure 400 {object} errors.ErrorResponse "Bad request"
// @Failure 401 {object} errors.ErrorResponse "Unauthorized"
// @Failure 404 {object} errors.ErrorResponse "Order not found"
// @Failure 500 {object} errors.ErrorResponse "Internal server error"
// @Router /api/v1/pipeline/orders/{id}/status [get]
// @Security BearerAuth
func (h *OrderPipelineHandler) GetOrderPipelineStatus(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		appErr := errors.NewValidationError("Order ID is required")
		middleware.AbortWithError(c, appErr)
		return
	}

	// Extract user ID from JWT claims for authorization
	claims, err := extractClaims(c)
	if err != nil {
		h.logger.Error("Failed to extract claims for pipeline status", "error", err)
		appErr := errors.NewUnauthorizedError("Invalid authentication token")
		middleware.AbortWithError(c, appErr)
		return
	}

	h.logger.Debug("Getting order pipeline status", "order_id", orderID, "user_id", claims.UserID)

	// TODO: Implement actual pipeline status retrieval from Redis/DB
	// For now, return a mock response
	mockResult := &services.OrderPipelineResult{
		Order: &services.OrderResponse{
			ID:     orderID,
			UserID: claims.UserID,
			Status: "completed",
		},
		ProcessingTime: 45 * time.Second,
		Status:         services.PipelineStatusCompleted,
		Notifications: []string{
			"order_confirmation_sent",
			"payment_confirmation_sent",
			"fulfillment_notification_sent",
		},
	}

	h.logger.Debug("Retrieved order pipeline status", "order_id", orderID, "status", mockResult.Status)

	c.JSON(http.StatusOK, mockResult)
}

// Helper function to safely extract order ID from pipeline result
func getOrderIDFromResult(result *services.OrderPipelineResult) string {
	if result != nil && result.Order != nil {
		return result.Order.ID
	}
	return "unknown"
}

// extractClaims extracts JWT claims from gin context
func extractClaims(c *gin.Context) (*jwt.Claims, error) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, errors.NewUnauthorizedError("Claims not found in context")
	}

	jwtClaims, ok := claims.(*jwt.Claims)
	if !ok {
		return nil, errors.NewUnauthorizedError("Invalid claims type in context")
	}

	return jwtClaims, nil
}
