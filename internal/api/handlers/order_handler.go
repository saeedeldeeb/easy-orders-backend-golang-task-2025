package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/services"
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
// @Description Create a new order with items
// @Tags orders
// @Accept json
// @Produce json
// @Param order body services.CreateOrderRequest true "Order details"
// @Success 201 {object} map[string]interface{} "Order created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "User or product not found"
// @Failure 409 {object} map[string]interface{} "Insufficient stock"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	h.logger.Debug("Creating order via API")

	var req services.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind order creation request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

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
// @Success 200 {object} map[string]interface{} "Order details"
// @Failure 400 {object} map[string]interface{} "Invalid order ID"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /orders/{id} [get]
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	h.logger.Debug("Getting order via API", "id", orderID)

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order ID is required",
		})
		return
	}

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

// UpdateOrderStatus godoc
// @Summary Update order status
// @Description Update the status of an existing order
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param status body object{status=string} true "New order status"
// @Success 200 {object} map[string]interface{} "Order status updated"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 409 {object} map[string]interface{} "Invalid status transition"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /orders/{id}/status [patch]
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	h.logger.Debug("Updating order status via API", "id", orderID)

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order ID is required",
		})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind order status update request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate status - basic validation for now
	status := models.OrderStatus(req.Status)
	validStatuses := []models.OrderStatus{
		models.OrderStatusPending,
		models.OrderStatusConfirmed,
		models.OrderStatusPaid,
		models.OrderStatusShipped,
		models.OrderStatusDelivered,
		models.OrderStatusCancelled,
	}

	isValid := false
	for _, validStatus := range validStatuses {
		if status == validStatus {
			isValid = true
			break
		}
	}

	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid order status",
		})
		return
	}

	// Call service
	order, err := h.orderService.UpdateOrderStatus(c.Request.Context(), orderID, status)
	if err != nil {
		h.logger.Error("Failed to update order status", "error", err, "id", orderID, "status", req.Status)

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
			return
		}

		if strings.Contains(err.Error(), "cannot transition") {
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update order status",
		})
		return
	}

	h.logger.Info("Order status updated successfully via API", "id", orderID, "new_status", req.Status)
	c.JSON(http.StatusOK, gin.H{
		"message": "Order status updated successfully",
		"data":    order,
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
	orderID := c.Param("id")
	h.logger.Debug("Cancelling order via API", "id", orderID)

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order ID is required",
		})
		return
	}

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
// @Param offset query int false "Offset for pagination" default(0)
// @Param limit query int false "Limit for pagination" default(10)
// @Param status query string false "Filter by order status"
// @Success 200 {object} map[string]interface{} "List of orders"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /orders [get]
func (h *OrderHandler) ListOrders(c *gin.Context) {
	h.logger.Debug("Listing orders via API")

	// Parse query parameters
	var req services.ListOrdersRequest

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			req.Offset = offset
		}
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	// Parse status filter
	if status := c.Query("status"); status != "" {
		req.Status = models.OrderStatus(status)
	}

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

// GetUserOrders godoc
// @Summary Get user orders
// @Description Get all orders for a specific user
// @Tags orders
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param offset query int false "Offset for pagination" default(0)
// @Param limit query int false "Limit for pagination" default(10)
// @Param status query string false "Filter by order status"
// @Success 200 {object} map[string]interface{} "User orders"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /users/{user_id}/orders [get]
func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userID := c.Param("user_id")
	h.logger.Debug("Getting user orders via API", "user_id", userID)

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	// Parse query parameters
	var req services.ListOrdersRequest

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			req.Offset = offset
		}
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	// Parse status filter
	if status := c.Query("status"); status != "" {
		req.Status = models.OrderStatus(status)
	}

	// Call service
	response, err := h.orderService.GetUserOrders(c.Request.Context(), userID, req)
	if err != nil {
		h.logger.Error("Failed to get user orders", "error", err, "user_id", userID)

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user orders",
		})
		return
	}

	h.logger.Debug("User orders retrieved successfully via API", "user_id", userID, "count", len(response.Orders))
	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}
