package handlers

import (
	"net/http"
	"strings"

	"easy-orders-backend/internal/api/middleware"
	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// OrderHandler handles order-related HTTP requests
type OrderHandler struct {
	orderService services.OrderService
	logger       *logger.Logger
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderService services.OrderService, logger *logger.Logger) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		logger:       logger,
	}
}

// CreateOrder godoc
// @Summary Create a new order
// @Description Create a new order with items. User ID is automatically extracted from the JWT token.
// @Tags orders
// @Accept json
// @Produce json
// @Param order body services.CreateOrderRequest true "Order details (user_id is extracted from JWT, not request body)"
// @Success 201 {object} object{message=string,data=services.OrderResponse} "Order created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User authentication failed"
// @Failure 404 {object} map[string]interface{} "User or product not found"
// @Failure 409 {object} map[string]interface{} "Insufficient stock"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	h.logger.Debug("Creating order via API")

	// Get validated request from context
	validatedReq, exists := middleware.GetValidatedRequest(c)
	if !exists {
		h.logger.Error("Validated request not found in context")
		appErr := errors.NewValidationError("Request validation failed")
		middleware.AbortWithError(c, appErr)
		return
	}

	// Type asserts to the expected request type
	req := *validatedReq.(*services.CreateOrderRequest)

	// Extract user ID from JWT context (authenticated user)
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		h.logger.Error("User ID not found in context")
		appErr := errors.NewUnauthorizedError("User authentication failed")
		middleware.AbortWithError(c, appErr)
		return
	}

	// Override UserID with authenticated user's ID for security
	req.UserID = userID
	h.logger.Debug("Using authenticated user ID for order", "user_id", userID)

	// Call service
	order, err := h.orderService.CreateOrder(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to create order", "error", err, "user_id", req.UserID)

		// Handle specific error types
		if strings.Contains(err.Error(), "not found") {
			if strings.Contains(err.Error(), "user") {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "User not found",
				})
			} else if strings.Contains(err.Error(), "product") {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "One or more products not found",
				})
			} else {
				c.JSON(http.StatusNotFound, gin.H{
					"error": err.Error(),
				})
			}
			return
		}

		if strings.Contains(err.Error(), "insufficient stock") || strings.Contains(err.Error(), "not available") {
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
			return
		}

		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create order",
		})
		return
	}

	h.logger.Info("Order created successfully via API", "id", order.ID, "user_id", req.UserID, "total", order.Total)
	c.JSON(http.StatusCreated, gin.H{
		"message": "Order created successfully",
		"data":    order,
	})
}

// GetOrder godoc
// @Summary Get order by ID
// @Description Retrieve order details by order ID
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} object{data=services.OrderResponse} "Order details"
// @Failure 400 {object} map[string]interface{} "Invalid order ID"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /orders/{id} [get]
func (h *OrderHandler) GetOrder(c *gin.Context) {
	// Middleware does path parameter validation
	orderID := c.Param("id")
	h.logger.Debug("Getting order via API", "id", orderID)

	// Call service
	order, err := h.orderService.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get order", "error", err, "id", orderID)

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get order",
		})
		return
	}

	h.logger.Debug("Order retrieved successfully via API", "id", orderID)
	c.JSON(http.StatusOK, gin.H{
		"data": order,
	})
}

// GetOrderStatus godoc
// @Summary Get order status
// @Description Retrieve only the status of an order by order ID
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} object{status=models.OrderStatus} "Order status"
// @Failure 400 {object} map[string]interface{} "Invalid order ID"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /orders/{id}/status [get]
func (h *OrderHandler) GetOrderStatus(c *gin.Context) {
	// Middleware does path parameter validation
	orderID := c.Param("id")
	h.logger.Debug("Getting order status via API", "id", orderID)

	// Call service to get order
	order, err := h.orderService.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get order status", "error", err, "id", orderID)

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get order status",
		})
		return
	}

	h.logger.Debug("Order status retrieved successfully via API", "id", orderID, "status", order.Status)
	c.JSON(http.StatusOK, gin.H{
		"status": order.Status,
	})
}

// CancelOrder godoc
// @Summary Cancel order
// @Description Cancel an existing order
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} map[string]interface{} "Order cancelled successfully"
// @Failure 400 {object} map[string]interface{} "Invalid order ID"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 409 {object} map[string]interface{} "Order cannot be cancelled"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /orders/{id}/cancel [patch]
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	// Path parameter validation is done by middleware
	orderID := c.Param("id")
	h.logger.Debug("Cancelling order via API", "id", orderID)

	// Call service
	err := h.orderService.CancelOrder(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to cancel order", "error", err, "id", orderID)

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
			return
		}

		if strings.Contains(err.Error(), "cannot be cancelled") {
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to cancel order",
		})
		return
	}

	h.logger.Info("Order cancelled successfully via API", "id", orderID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Order cancelled successfully",
	})
}

// ListOrders godoc
// @Summary List orders
// @Description Get a paginated list of orders with optional status filter
// @Tags orders
// @Accept json
// @Produce json
// @Param page query int false "Page number for pagination" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param status query string false "Filter by order status"
// @Success 200 {object} object{data=services.ListOrdersResponse} "List of orders"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /orders [get]
func (h *OrderHandler) ListOrders(c *gin.Context) {
	h.logger.Debug("Listing orders via API")

	// Get validated query from context
	validatedQuery, exists := middleware.GetValidatedQuery(c)
	if !exists {
		h.logger.Error("Validated query not found in context")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request validation failed"})
		return
	}

	// Type asserts to the expected request type
	req := *validatedQuery.(*services.ListOrdersRequest)

	// Call service
	response, err := h.orderService.ListOrders(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to list orders", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list orders",
		})
		return
	}

	h.logger.Debug("Orders listed successfully via API", "count", len(response.Orders))
	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}
