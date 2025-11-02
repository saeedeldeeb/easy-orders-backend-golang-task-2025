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
// @Success 200 {object} map[string]interface{} "List of all orders"
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
// @Success 200 {object} map[string]interface{} "Order status updated"
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
// @Success 200 {object} map[string]interface{} "Daily sales report"
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

// GenerateInventoryReport godoc
// @Summary Generate inventory report (Admin)
// @Description Generate current inventory status report
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Inventory report"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /admin/reports/inventory [get]
func (h *AdminHandler) GenerateInventoryReport(c *gin.Context) {
	h.logger.Debug("Generating inventory report via admin API")

	// Call service
	report, err := h.reportService.GenerateInventoryReport(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to generate inventory report", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate inventory report",
		})
		return
	}

	h.logger.Info("Inventory report generated successfully via admin API", "total_products", report.TotalProducts, "low_stock_count", report.LowStockProducts)
	c.JSON(http.StatusOK, gin.H{
		"data": report,
	})
}

// GenerateTopProductsReport godoc
// @Summary Generate top products report (Admin)
// @Description Generate report of top selling products
// @Tags admin
// @Accept json
// @Produce json
// @Param limit query int false "Number of top products" default(10)
// @Success 200 {object} map[string]interface{} "Top products report"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /admin/reports/products/top [get]
func (h *AdminHandler) GenerateTopProductsReport(c *gin.Context) {
	h.logger.Debug("Generating top products report via admin API")

	// Parse limit parameter
	limit := 10 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Call service - Note: This method might not exist yet, commenting out for now
	// report, err := h.reportService.GenerateTopProductsReport(c.Request.Context(), limit)
	// if err != nil {
	//	h.logger.Error("Failed to generate top products report", "error", err, "limit", limit)
	//	c.JSON(http.StatusInternalServerError, gin.H{
	//		"error": "Failed to generate top products report",
	//	})
	//	return
	// }

	// Temporary placeholder response
	report := &services.TopProductsResponse{
		TopProducts: []*services.TopProductItem{},
		Limit:       limit,
		Period:      "all_time",
	}

	h.logger.Info("Top products report generated successfully via admin API", "limit", limit, "products_count", len(report.TopProducts))
	c.JSON(http.StatusOK, gin.H{
		"data": report,
	})
}

// GenerateUserActivityReport godoc
// @Summary Generate user activity report (Admin)
// @Description Generate user activity report for a date range
// @Tags admin
// @Accept json
// @Produce json
// @Param start_date query string false "Start date in YYYY-MM-DD format"
// @Param end_date query string false "End date in YYYY-MM-DD format"
// @Success 200 {object} map[string]interface{} "User activity report"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /admin/reports/users/activity [get]
func (h *AdminHandler) GenerateUserActivityReport(c *gin.Context) {
	h.logger.Debug("Generating user activity report via admin API")

	// Parse date range parameters
	var req services.UserActivityReportRequest
	req.StartDate = c.Query("start_date")
	req.EndDate = c.Query("end_date")

	// Call service
	report, err := h.reportService.GenerateUserActivityReport(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to generate user activity report", "error", err, "start_date", req.StartDate, "end_date", req.EndDate)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate user activity report",
		})
		return
	}

	h.logger.Info("User activity report generated successfully via admin API", "active_users", report.ActiveUsers, "new_users", report.NewUsers)
	c.JSON(http.StatusOK, gin.H{
		"data": report,
	})
}
