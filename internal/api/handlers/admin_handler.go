package handlers

import (
	"net/http"
	"strings"

	"easy-orders-backend/internal/api/middleware"
	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/errors"
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
// @Param page query int false "Page number for pagination" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param status query string false "Filter by order status"
// @Success 200 {object} object{data=services.ListOrdersResponse} "List of all orders"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /admin/orders [get]
func (h *AdminHandler) GetAllOrders(c *gin.Context) {
	h.logger.Debug("Getting all orders via admin API")

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
	// Path parameter validation is done by middleware
	orderID := c.Param("id")
	h.logger.Debug("Updating order status via admin API", "id", orderID)

	// Get validated request from context
	validatedReq, exists := middleware.GetValidatedRequest(c)
	if !exists {
		h.logger.Error("Validated request not found in context")
		appErr := errors.NewValidationError("Request validation failed")
		middleware.AbortWithError(c, appErr)
		return
	}

	// Type asserts to the expected request type
	req := *validatedReq.(*services.UpdateStatusRequest)

	// Call service (cast string to OrderStatus type)
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
// @Router /admin/reports/daily [get]
func (h *AdminHandler) GenerateDailySalesReport(c *gin.Context) {
	h.logger.Debug("Generating daily sales report via admin API")

	// Get validated query from context
	validatedQuery, exists := middleware.GetValidatedQuery(c)
	if !exists {
		h.logger.Error("Validated query not found in context")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request validation failed"})
		return
	}

	// Type asserts to the expected request type
	req := *validatedQuery.(*services.DailySalesReportQuery)

	// Call service
	report, err := h.reportService.GenerateDailySalesReport(c.Request.Context(), req.Date)
	if err != nil {
		h.logger.Error("Failed to generate daily sales report", "error", err, "date", req.Date)

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
