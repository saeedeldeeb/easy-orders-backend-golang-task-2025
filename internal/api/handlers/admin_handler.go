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

// AdminHandler handles admin-related HTTP requests
type AdminHandler struct {
	orderService  services.OrderService
	reportService services.ReportService
	logger        *logger.Logger
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	orderService services.OrderService,
	reportService services.ReportService,
	logger *logger.Logger,
) *AdminHandler {
	return &AdminHandler{
		orderService:  orderService,
		reportService: reportService,
		logger:        logger,
	}
}

// GetAllOrders godoc
// @Summary Get all orders (Admin)
// @Description Get a paginated list of all orders (Admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Param offset query int false "Offset for pagination" default(0)
// @Param limit query int false "Limit for pagination" default(10)
// @Param status query string false "Filter by order status"
// @Success 200 {object} object{data=services.ListOrdersResponse} "List of all orders"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /admin/orders [get]
func (h *AdminHandler) GetAllOrders(c *gin.Context) {
	h.logger.Debug("Getting all orders via admin API")

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
		h.logger.Error("Failed to get all orders", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get orders",
		})
		return
	}

	h.logger.Debug("All orders retrieved successfully via admin API", "count", len(response.Orders))
	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// UpdateOrderStatus godoc
// @Summary Update order status (Admin)
// @Description Update order status as admin
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param status body object{status=string} true "New order status"
// @Success 200 {object} object{message=string,data=services.OrderResponse} "Order status updated"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 409 {object} map[string]interface{} "Invalid status transition"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /admin/orders/{id}/status [patch]
func (h *AdminHandler) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	h.logger.Debug("Updating order status via admin API", "id", orderID)

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

	// Call service (reuse order service functionality)
	order, err := h.orderService.UpdateOrderStatus(c.Request.Context(), orderID, models.OrderStatus(req.Status))
	if err != nil {
		h.logger.Error("Failed to update order status via admin", "error", err, "id", orderID, "status", req.Status)

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

	h.logger.Info("Order status updated successfully via admin API", "id", orderID, "new_status", req.Status)
	c.JSON(http.StatusOK, gin.H{
		"message": "Order status updated successfully",
		"data":    order,
	})
}

// GenerateDailySalesReport godoc
// @Summary Generate daily sales report (Admin)
// @Description Generate sales report for a specific date
// @Tags admin
// @Accept json
// @Produce json
// @Param date query string false "Date in YYYY-MM-DD format"
// @Success 200 {object} object{data=services.SalesReportResponse} "Daily sales report"
// @Failure 400 {object} map[string]interface{} "Invalid date format"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /admin/reports/sales/daily [get]
func (h *AdminHandler) GenerateDailySalesReport(c *gin.Context) {
	h.logger.Debug("Generating daily sales report via admin API")

	date := c.Query("date") // Format: YYYY-MM-DD

	// Call service
	report, err := h.reportService.GenerateDailySalesReport(c.Request.Context(), date)
	if err != nil {
		h.logger.Error("Failed to generate daily sales report", "error", err, "date", date)

		if strings.Contains(err.Error(), "invalid date format") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid date format, use YYYY-MM-DD",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate sales report",
		})
		return
	}

	h.logger.Info("Daily sales report generated successfully via admin API", "date", report.Date, "total_sales", report.TotalSales)
	c.JSON(http.StatusOK, gin.H{
		"data": report,
	})
}
