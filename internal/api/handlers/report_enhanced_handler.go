package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ReportEnhancedHandler handles enhanced report generation endpoints
type ReportEnhancedHandler struct {
	enhancedService *services.EnhancedReportService
	logger          *logger.Logger
}

// NewReportEnhancedHandler creates a new enhanced report handler
func NewReportEnhancedHandler(enhancedService *services.EnhancedReportService, logger *logger.Logger) *ReportEnhancedHandler {
	return &ReportEnhancedHandler{
		enhancedService: enhancedService,
		logger:          logger,
	}
}

// RegisterRoutes registers enhanced report routes
func (h *ReportEnhancedHandler) RegisterRoutes(router *gin.RouterGroup) {
	reports := router.Group("/reports")
	{
		// Sales reports
		reports.GET("/sales/daily", h.GenerateDailySalesReport)
		reports.GET("/sales/weekly", h.GenerateWeeklySalesReport)
		reports.GET("/sales/monthly", h.GenerateMonthlySalesReport)
		reports.GET("/sales/top-products", h.GenerateTopProductsReport)

		// Inventory reports
		reports.GET("/inventory/low-stock", h.GenerateLowStockReport)
		reports.GET("/inventory/value", h.GenerateInventoryValueReport)

		// Customer reports
		reports.GET("/customers/activity", h.GenerateCustomerActivityReport)
		reports.GET("/customers/orders", h.GenerateOrderAnalyticsReport)

		// Generic report generation
		reports.POST("/generate", h.GenerateCustomReport)
		reports.POST("/generate/async", h.GenerateCustomReportAsync)
		reports.POST("/generate/batch", h.GenerateBatchReports)

		// Report management
		reports.GET("/types", h.GetSupportedTypes)
		reports.GET("/metrics", h.GetReportMetrics)
		reports.GET("/cache/stats", h.GetCacheStats)
		reports.POST("/cache/flush", h.FlushCache)

		// Background job management
		reports.POST("/jobs/submit", h.SubmitReportJob)
	}
}

// GenerateDailySalesReport generates a daily sales report
func (h *ReportEnhancedHandler) GenerateDailySalesReport(c *gin.Context) {
	date := c.Query("date")
	async := c.Query("async") == "true"

	h.logger.Info("Generating daily sales report", "date", date, "async", async)

	result, err := h.enhancedService.GenerateDailySalesReport(c.Request.Context(), date, async)
	if err != nil {
		h.logger.Error("Failed to generate daily sales report", "error", err)
		c.Error(errors.NewInternalError("Failed to generate daily sales report", err))
		return
	}

	statusCode := http.StatusOK
	if async && result.Status == "pending" {
		statusCode = http.StatusAccepted
	}

	c.JSON(statusCode, gin.H{
		"success": true,
		"result":  result,
	})
}

// GenerateWeeklySalesReport generates a weekly sales report
func (h *ReportEnhancedHandler) GenerateWeeklySalesReport(c *gin.Context) {
	weekStart := c.Query("week_start")
	async := c.Query("async") == "true"

	h.logger.Info("Generating weekly sales report", "week_start", weekStart, "async", async)

	result, err := h.enhancedService.GenerateWeeklySalesReport(c.Request.Context(), weekStart, async)
	if err != nil {
		h.logger.Error("Failed to generate weekly sales report", "error", err)
		c.Error(errors.NewInternalError("Failed to generate weekly sales report", err))
		return
	}

	statusCode := http.StatusOK
	if async && result.Status == "pending" {
		statusCode = http.StatusAccepted
	}

	c.JSON(statusCode, gin.H{
		"success": true,
		"result":  result,
	})
}

// GenerateMonthlySalesReport generates a monthly sales report
func (h *ReportEnhancedHandler) GenerateMonthlySalesReport(c *gin.Context) {
	yearStr := c.Query("year")
	monthStr := c.Query("month")
	async := c.Query("async") == "true"

	year := time.Now().Year()
	month := int(time.Now().Month())

	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}
	if monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil {
			month = m
		}
	}

	h.logger.Info("Generating monthly sales report",
		"year", year,
		"month", month,
		"async", async)

	result, err := h.enhancedService.GenerateMonthlySalesReport(c.Request.Context(), year, month, async)
	if err != nil {
		h.logger.Error("Failed to generate monthly sales report", "error", err)
		c.Error(errors.NewInternalError("Failed to generate monthly sales report", err))
		return
	}

	statusCode := http.StatusOK
	if async && result.Status == "pending" {
		statusCode = http.StatusAccepted
	}

	c.JSON(statusCode, gin.H{
		"success": true,
		"result":  result,
	})
}

// GenerateTopProductsReport generates a top products report
func (h *ReportEnhancedHandler) GenerateTopProductsReport(c *gin.Context) {
	period := c.DefaultQuery("period", "last_30_days")
	limitStr := c.DefaultQuery("limit", "20")
	async := c.Query("async") == "true"

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	h.logger.Info("Generating top products report",
		"period", period,
		"limit", limit,
		"async", async)

	result, err := h.enhancedService.GenerateTopProductsReport(c.Request.Context(), period, limit, async)
	if err != nil {
		h.logger.Error("Failed to generate top products report", "error", err)
		c.Error(errors.NewInternalError("Failed to generate top products report", err))
		return
	}

	statusCode := http.StatusOK
	if async && result.Status == "pending" {
		statusCode = http.StatusAccepted
	}

	c.JSON(statusCode, gin.H{
		"success": true,
		"result":  result,
	})
}

// GenerateLowStockReport generates a low stock alert report
func (h *ReportEnhancedHandler) GenerateLowStockReport(c *gin.Context) {
	thresholdStr := c.DefaultQuery("threshold", "10")
	async := c.Query("async") == "true"

	threshold, err := strconv.Atoi(thresholdStr)
	if err != nil {
		threshold = 10
	}

	h.logger.Info("Generating low stock report",
		"threshold", threshold,
		"async", async)

	result, err := h.enhancedService.GenerateLowStockReport(c.Request.Context(), threshold, async)
	if err != nil {
		h.logger.Error("Failed to generate low stock report", "error", err)
		c.Error(errors.NewInternalError("Failed to generate low stock report", err))
		return
	}

	statusCode := http.StatusOK
	if async && result.Status == "pending" {
		statusCode = http.StatusAccepted
	}

	c.JSON(statusCode, gin.H{
		"success": true,
		"result":  result,
	})
}

// GenerateInventoryValueReport generates an inventory valuation report
func (h *ReportEnhancedHandler) GenerateInventoryValueReport(c *gin.Context) {
	async := c.Query("async") == "true"

	h.logger.Info("Generating inventory value report", "async", async)

	result, err := h.enhancedService.GenerateInventoryValueReport(c.Request.Context(), async)
	if err != nil {
		h.logger.Error("Failed to generate inventory value report", "error", err)
		c.Error(errors.NewInternalError("Failed to generate inventory value report", err))
		return
	}

	statusCode := http.StatusOK
	if async && result.Status == "pending" {
		statusCode = http.StatusAccepted
	}

	c.JSON(statusCode, gin.H{
		"success": true,
		"result":  result,
	})
}

// GenerateCustomerActivityReport generates a customer activity report
func (h *ReportEnhancedHandler) GenerateCustomerActivityReport(c *gin.Context) {
	period := c.DefaultQuery("period", "last_30_days")
	async := c.Query("async") == "true"

	h.logger.Info("Generating customer activity report",
		"period", period,
		"async", async)

	result, err := h.enhancedService.GenerateCustomerActivityReport(c.Request.Context(), period, async)
	if err != nil {
		h.logger.Error("Failed to generate customer activity report", "error", err)
		c.Error(errors.NewInternalError("Failed to generate customer activity report", err))
		return
	}

	statusCode := http.StatusOK
	if async && result.Status == "pending" {
		statusCode = http.StatusAccepted
	}

	c.JSON(statusCode, gin.H{
		"success": true,
		"result":  result,
	})
}

// GenerateOrderAnalyticsReport generates an order analytics report
func (h *ReportEnhancedHandler) GenerateOrderAnalyticsReport(c *gin.Context) {
	period := c.DefaultQuery("period", "last_30_days")
	async := c.Query("async") == "true"

	h.logger.Info("Generating order analytics report",
		"period", period,
		"async", async)

	// For now, we'll use customer activity report as a placeholder
	result, err := h.enhancedService.GenerateCustomerActivityReport(c.Request.Context(), period, async)
	if err != nil {
		h.logger.Error("Failed to generate order analytics report", "error", err)
		c.Error(errors.NewInternalError("Failed to generate order analytics report", err))
		return
	}

	statusCode := http.StatusOK
	if async && result.Status == "pending" {
		statusCode = http.StatusAccepted
	}

	c.JSON(statusCode, gin.H{
		"success": true,
		"result":  result,
	})
}

// CustomReportRequest represents a custom report generation request
type CustomReportRequest struct {
	Type       string                 `json:"type" binding:"required"`
	Format     string                 `json:"format"`
	Parameters map[string]interface{} `json:"parameters"`
}

// GenerateCustomReport generates a custom report synchronously
func (h *ReportEnhancedHandler) GenerateCustomReport(c *gin.Context) {
	var req CustomReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if req.Format == "" {
		req.Format = "json"
	}
	if req.Parameters == nil {
		req.Parameters = make(map[string]interface{})
	}

	h.logger.Info("Generating custom report",
		"type", req.Type,
		"format", req.Format)

	result, err := h.enhancedService.GenerateReportSync(c.Request.Context(), req.Type, req.Format, req.Parameters)
	if err != nil {
		h.logger.Error("Failed to generate custom report", "error", err, "type", req.Type)
		c.Error(errors.NewInternalError("Failed to generate custom report", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

// GenerateCustomReportAsync generates a custom report asynchronously
func (h *ReportEnhancedHandler) GenerateCustomReportAsync(c *gin.Context) {
	var req CustomReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if req.Format == "" {
		req.Format = "json"
	}
	if req.Parameters == nil {
		req.Parameters = make(map[string]interface{})
	}

	h.logger.Info("Generating custom report asynchronously",
		"type", req.Type,
		"format", req.Format)

	result, err := h.enhancedService.GenerateReportAsync(c.Request.Context(), req.Type, req.Format, req.Parameters)
	if err != nil {
		h.logger.Error("Failed to generate custom report async", "error", err, "type", req.Type)
		c.Error(errors.NewInternalError("Failed to generate custom report", err))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"result":  result,
	})
}

// BatchReportRequest represents a batch report generation request
type BatchReportRequest struct {
	Reports []CustomReportRequest `json:"reports" binding:"required"`
}

// GenerateBatchReports generates multiple reports in batch
func (h *ReportEnhancedHandler) GenerateBatchReports(c *gin.Context) {
	var req BatchReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	h.logger.Info("Generating batch reports", "count", len(req.Reports))

	var results []interface{}
	var errors []string

	for i, reportReq := range req.Reports {
		if reportReq.Format == "" {
			reportReq.Format = "json"
		}
		if reportReq.Parameters == nil {
			reportReq.Parameters = make(map[string]interface{})
		}

		result, err := h.enhancedService.GenerateReportAsync(c.Request.Context(), reportReq.Type, reportReq.Format, reportReq.Parameters)
		if err != nil {
			h.logger.Error("Failed to generate batch report", "error", err, "index", i, "type", reportReq.Type)
			errors = append(errors, fmt.Sprintf("Report %d (%s): %s", i+1, reportReq.Type, err.Error()))
		} else {
			results = append(results, result)
		}
	}

	statusCode := http.StatusOK
	if len(errors) > 0 {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, gin.H{
		"success":         len(errors) == 0,
		"results":         results,
		"errors":          errors,
		"total_requested": len(req.Reports),
		"successful":      len(results),
		"failed":          len(errors),
	})
}

// GetSupportedTypes returns all supported report types
func (h *ReportEnhancedHandler) GetSupportedTypes(c *gin.Context) {
	types := h.enhancedService.GetSupportedTypes()

	c.JSON(http.StatusOK, gin.H{
		"supported_types": types,
		"count":           len(types),
	})
}

// GetReportMetrics returns report generation metrics
func (h *ReportEnhancedHandler) GetReportMetrics(c *gin.Context) {
	metrics := h.enhancedService.GetReportMetrics()

	c.JSON(http.StatusOK, gin.H{
		"metrics": metrics,
	})
}

// GetCacheStats returns cache statistics
func (h *ReportEnhancedHandler) GetCacheStats(c *gin.Context) {
	stats := h.enhancedService.GetCacheStats()

	c.JSON(http.StatusOK, gin.H{
		"cache_stats": stats,
	})
}

// FlushCache clears all cached reports
func (h *ReportEnhancedHandler) FlushCache(c *gin.Context) {
	h.enhancedService.FlushCache()

	h.logger.Info("Report cache flushed via API")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache flushed successfully",
	})
}

// ReportJobRequest represents a background report job request
type ReportJobRequest struct {
	Type       string                 `json:"type" binding:"required"`
	Format     string                 `json:"format"`
	Parameters map[string]interface{} `json:"parameters"`
	Priority   int                    `json:"priority"`
}

// SubmitReportJob submits a report generation job to the background worker pool
func (h *ReportEnhancedHandler) SubmitReportJob(c *gin.Context) {
	var req ReportJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if req.Format == "" {
		req.Format = "json"
	}
	if req.Parameters == nil {
		req.Parameters = make(map[string]interface{})
	}
	if req.Priority == 0 {
		req.Priority = 5 // Normal priority
	}

	h.logger.Info("Submitting report job",
		"type", req.Type,
		"format", req.Format,
		"priority", req.Priority)

	result, err := h.enhancedService.SubmitReportJob(c.Request.Context(), req.Type, req.Format, req.Parameters, req.Priority)
	if err != nil {
		h.logger.Error("Failed to submit report job", "error", err, "type", req.Type)
		c.Error(errors.NewInternalError("Failed to submit report job", err))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"job":     result,
	})
}
