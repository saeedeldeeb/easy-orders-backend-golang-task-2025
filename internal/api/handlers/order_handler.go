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
// @Success 201 {object} object{message=string,data=services.OrderResponse} "Order created successfully"
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
// @Success 200 {object} object{data=services.OrderResponse} "Order details"
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
	orderID := c.Param("id")
	h.logger.Debug("Getting order status via API", "id", orderID)

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order ID is required",
		})
		return
	}

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
// @Success 200 {object} object{data=services.ListOrdersResponse} "List of orders"
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
