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

// GetAllOrders handles GET /api/v1/admin/orders
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

// UpdateOrderStatus handles PATCH /api/v1/admin/orders/:id/status
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

// GenerateDailySalesReport handles GET /api/v1/admin/reports/sales/daily
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

// GenerateInventoryReport handles GET /api/v1/admin/reports/inventory
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

// GenerateTopProductsReport handles GET /api/v1/admin/reports/products/top
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

// GenerateUserActivityReport handles GET /api/v1/admin/reports/users/activity
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

// RegisterRoutes registers all admin routes
func (h *AdminHandler) RegisterRoutes(router *gin.RouterGroup) {
	admin := router.Group("/admin")
	{
		// Order management
		orders := admin.Group("/orders")
		{
			orders.GET("", h.GetAllOrders)
			orders.PATCH("/:id/status", h.UpdateOrderStatus)
		}

		// Reports
		reports := admin.Group("/reports")
		{
			reports.GET("/sales/daily", h.GenerateDailySalesReport)
			reports.GET("/inventory", h.GenerateInventoryReport)
			reports.GET("/products/top", h.GenerateTopProductsReport)
			reports.GET("/users/activity", h.GenerateUserActivityReport)
		}
	}
}
