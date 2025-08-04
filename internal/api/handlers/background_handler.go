package handlers

import (
	"net/http"
	"time"

	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// BackgroundHandler handles background job management endpoints
type BackgroundHandler struct {
	backgroundService *services.BackgroundService
	logger            *logger.Logger
}

// NewBackgroundHandler creates a new background handler
func NewBackgroundHandler(backgroundService *services.BackgroundService, logger *logger.Logger) *BackgroundHandler {
	return &BackgroundHandler{
		backgroundService: backgroundService,
		logger:            logger,
	}
}

// RegisterRoutes registers background job routes
func (h *BackgroundHandler) RegisterRoutes(router *gin.RouterGroup) {
	bg := router.Group("/background")
	{
		// Job submission endpoints
		bg.POST("/jobs/report", h.SubmitReportJob)
		bg.POST("/jobs/notification", h.SubmitNotificationJob)
		bg.POST("/jobs/audit", h.SubmitAuditJob)
		bg.POST("/jobs/bulk", h.SubmitBulkJob)
		bg.POST("/jobs/integration", h.SubmitIntegrationJob)

		// Monitoring endpoints
		bg.GET("/metrics", h.GetMetrics)
		bg.GET("/metrics/:pool", h.GetPoolMetrics)
	}
}

// SubmitReportJobRequest represents a report job submission request
type SubmitReportJobRequest struct {
	ReportType string                 `json:"report_type" binding:"required"`
	Parameters map[string]interface{} `json:"parameters"`
}

// SubmitReportJob submits a report generation job
func (h *BackgroundHandler) SubmitReportJob(c *gin.Context) {
	var req SubmitReportJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if err := h.backgroundService.SubmitReportJob(req.ReportType, req.Parameters); err != nil {
		h.logger.Error("Failed to submit report job", "error", err, "type", req.ReportType)
		c.Error(errors.NewInternalError("Failed to submit report job", err))
		return
	}

	h.logger.Info("Report job submitted", "type", req.ReportType)
	c.JSON(http.StatusAccepted, gin.H{
		"message":     "Report job submitted successfully",
		"report_type": req.ReportType,
	})
}

// SubmitNotificationJobRequest represents a notification job submission request
type SubmitNotificationJobRequest struct {
	RecipientType    string                 `json:"recipient_type" binding:"required"`
	RecipientID      string                 `json:"recipient_id" binding:"required"`
	NotificationType string                 `json:"notification_type" binding:"required"`
	Template         string                 `json:"template" binding:"required"`
	Data             map[string]interface{} `json:"data"`
}

// SubmitNotificationJob submits a notification job
func (h *BackgroundHandler) SubmitNotificationJob(c *gin.Context) {
	var req SubmitNotificationJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if err := h.backgroundService.SubmitNotificationJob(
		req.RecipientType,
		req.RecipientID,
		req.NotificationType,
		req.Template,
		req.Data,
	); err != nil {
		h.logger.Error("Failed to submit notification job", "error", err)
		c.Error(errors.NewInternalError("Failed to submit notification job", err))
		return
	}

	h.logger.Info("Notification job submitted",
		"recipient_type", req.RecipientType,
		"notification_type", req.NotificationType)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Notification job submitted successfully",
	})
}

// SubmitAuditJobRequest represents an audit job submission request
type SubmitAuditJobRequest struct {
	EntityType string                 `json:"entity_type" binding:"required"`
	EntityID   string                 `json:"entity_id" binding:"required"`
	Action     string                 `json:"action" binding:"required"`
	UserID     string                 `json:"user_id" binding:"required"`
	Changes    map[string]interface{} `json:"changes"`
}

// SubmitAuditJob submits an audit processing job
func (h *BackgroundHandler) SubmitAuditJob(c *gin.Context) {
	var req SubmitAuditJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if err := h.backgroundService.SubmitAuditJob(
		req.EntityType,
		req.EntityID,
		req.Action,
		req.UserID,
		req.Changes,
	); err != nil {
		h.logger.Error("Failed to submit audit job", "error", err)
		c.Error(errors.NewInternalError("Failed to submit audit job", err))
		return
	}

	h.logger.Info("Audit job submitted",
		"entity_type", req.EntityType,
		"action", req.Action)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Audit job submitted successfully",
	})
}

// SubmitBulkJobRequest represents a bulk processing job submission request
type SubmitBulkJobRequest struct {
	OperationType string                 `json:"operation_type" binding:"required"`
	EntityIDs     []string               `json:"entity_ids" binding:"required"`
	BatchSize     int                    `json:"batch_size"`
	Parameters    map[string]interface{} `json:"parameters"`
}

// SubmitBulkJob submits a bulk processing job
func (h *BackgroundHandler) SubmitBulkJob(c *gin.Context) {
	var req SubmitBulkJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if req.BatchSize <= 0 {
		req.BatchSize = 50 // Default batch size
	}

	if err := h.backgroundService.SubmitBulkProcessingJob(
		req.OperationType,
		req.EntityIDs,
		req.BatchSize,
		req.Parameters,
	); err != nil {
		h.logger.Error("Failed to submit bulk job", "error", err)
		c.Error(errors.NewInternalError("Failed to submit bulk job", err))
		return
	}

	h.logger.Info("Bulk job submitted",
		"operation_type", req.OperationType,
		"entity_count", len(req.EntityIDs))

	c.JSON(http.StatusAccepted, gin.H{
		"message":      "Bulk job submitted successfully",
		"entity_count": len(req.EntityIDs),
	})
}

// SubmitIntegrationJobRequest represents an external integration job submission request
type SubmitIntegrationJobRequest struct {
	ServiceName string                 `json:"service_name" binding:"required"`
	Operation   string                 `json:"operation" binding:"required"`
	Payload     map[string]interface{} `json:"payload"`
	Timeout     int                    `json:"timeout"` // timeout in seconds
}

// SubmitIntegrationJob submits an external integration job
func (h *BackgroundHandler) SubmitIntegrationJob(c *gin.Context) {
	var req SubmitIntegrationJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	timeout := time.Duration(req.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second // Default timeout
	}

	if err := h.backgroundService.SubmitExternalIntegrationJob(
		req.ServiceName,
		req.Operation,
		req.Payload,
		timeout,
	); err != nil {
		h.logger.Error("Failed to submit integration job", "error", err)
		c.Error(errors.NewInternalError("Failed to submit integration job", err))
		return
	}

	h.logger.Info("Integration job submitted",
		"service", req.ServiceName,
		"operation", req.Operation)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Integration job submitted successfully",
	})
}

// GetMetrics returns metrics for all worker pools
func (h *BackgroundHandler) GetMetrics(c *gin.Context) {
	metrics := h.backgroundService.GetMetrics()

	c.JSON(http.StatusOK, gin.H{
		"worker_pools": metrics,
	})
}

// GetPoolMetrics returns metrics for a specific worker pool
func (h *BackgroundHandler) GetPoolMetrics(c *gin.Context) {
	poolName := c.Param("pool")

	metrics, err := h.backgroundService.GetPoolMetrics(poolName)
	if err != nil {
		c.Error(errors.NewNotFoundError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pool":    poolName,
		"metrics": metrics,
	})
}
