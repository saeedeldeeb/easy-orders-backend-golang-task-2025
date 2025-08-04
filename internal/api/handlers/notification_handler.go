package handlers

import (
	"net/http"
	"time"

	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/logger"
	"easy-orders-backend/pkg/notifications"

	"github.com/gin-gonic/gin"
)

// NotificationHandler handles advanced notification endpoints
type NotificationHandler struct {
	enhancedService *services.EnhancedNotificationService
	logger          *logger.Logger
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(enhancedService *services.EnhancedNotificationService, logger *logger.Logger) *NotificationHandler {
	return &NotificationHandler{
		enhancedService: enhancedService,
		logger:          logger,
	}
}

// RegisterRoutes registers notification routes
func (h *NotificationHandler) RegisterRoutes(router *gin.RouterGroup) {
	notif := router.Group("/notifications")
	{
		// Enhanced notification endpoints
		notif.POST("/dispatch", h.DispatchNotification)
		notif.POST("/bulk", h.SendBulkNotifications)
		notif.POST("/scheduled", h.SendScheduledNotification)

		// Notification management
		notif.GET("/metrics", h.GetMetrics)
		notif.GET("/channels", h.GetChannels)
		notif.GET("/templates", h.GetTemplates)

		// Specialized notifications
		notif.POST("/order-confirmation", h.SendOrderConfirmation)
		notif.POST("/order-shipped", h.SendOrderShipped)
		notif.POST("/payment-success", h.SendPaymentSuccess)
		notif.POST("/welcome", h.SendWelcome)
		notif.POST("/low-stock-alert", h.SendLowStockAlert)
	}
}

// DispatchNotificationRequest represents a notification dispatch request
type DispatchNotificationRequest struct {
	Type       string                 `json:"type" binding:"required"`
	Channel    string                 `json:"channel" binding:"required"`
	Recipient  string                 `json:"recipient" binding:"required"`
	Subject    string                 `json:"subject"`
	Body       string                 `json:"body"`
	TemplateID string                 `json:"template_id"`
	Data       map[string]interface{} `json:"data"`
	Priority   int                    `json:"priority"`
	MaxRetries int                    `json:"max_retries"`
}

// DispatchNotification dispatches a single notification
func (h *NotificationHandler) DispatchNotification(c *gin.Context) {
	var req DispatchNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	notification := notifications.NewNotification(
		notifications.NotificationType(req.Type),
		req.Channel,
		req.Recipient,
	)

	notification.Subject = req.Subject
	notification.Body = req.Body
	notification.TemplateID = req.TemplateID
	notification.Data = req.Data

	if req.Priority > 0 {
		notification.SetPriority(notifications.NotificationPriority(req.Priority))
	}

	if req.MaxRetries > 0 {
		notification.MaxRetries = req.MaxRetries
	}

	if err := h.enhancedService.DispatchNotification(notification); err != nil {
		h.logger.Error("Failed to dispatch notification", "error", err)
		c.Error(errors.NewInternalError("Failed to dispatch notification", err))
		return
	}

	h.logger.Info("Notification dispatched", "id", notification.ID, "type", req.Type)
	c.JSON(http.StatusAccepted, gin.H{
		"message":         "Notification dispatched successfully",
		"notification_id": notification.ID,
	})
}

// BulkNotificationRequest represents a bulk notification request
type BulkNotificationRequest struct {
	Notifications []DispatchNotificationRequest `json:"notifications" binding:"required"`
}

// SendBulkNotifications sends multiple notifications in batch
func (h *NotificationHandler) SendBulkNotifications(c *gin.Context) {
	var req BulkNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	notificationList := make([]*notifications.Notification, len(req.Notifications))
	for i, notifReq := range req.Notifications {
		notification := notifications.NewNotification(
			notifications.NotificationType(notifReq.Type),
			notifReq.Channel,
			notifReq.Recipient,
		)

		notification.Subject = notifReq.Subject
		notification.Body = notifReq.Body
		notification.TemplateID = notifReq.TemplateID
		notification.Data = notifReq.Data

		if notifReq.Priority > 0 {
			notification.SetPriority(notifications.NotificationPriority(notifReq.Priority))
		}

		notificationList[i] = notification
	}

	if err := h.enhancedService.SendBulkNotifications(notificationList); err != nil {
		h.logger.Error("Failed to send bulk notifications", "error", err)
		c.Error(errors.NewInternalError("Failed to send bulk notifications", err))
		return
	}

	h.logger.Info("Bulk notifications dispatched", "count", len(notificationList))
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Bulk notifications dispatched successfully",
		"count":   len(notificationList),
	})
}

// ScheduledNotificationRequest represents a scheduled notification request
type ScheduledNotificationRequest struct {
	DispatchNotificationRequest
	ScheduledAt string `json:"scheduled_at" binding:"required"` // RFC3339 format
}

// SendScheduledNotification schedules a notification for future delivery
func (h *NotificationHandler) SendScheduledNotification(c *gin.Context) {
	var req ScheduledNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		c.Error(errors.NewValidationError("Invalid scheduled_at format"))
		return
	}

	notification := notifications.NewNotification(
		notifications.NotificationType(req.Type),
		req.Channel,
		req.Recipient,
	)

	notification.Subject = req.Subject
	notification.Body = req.Body
	notification.TemplateID = req.TemplateID
	notification.Data = req.Data

	if req.Priority > 0 {
		notification.SetPriority(notifications.NotificationPriority(req.Priority))
	}

	if err := h.enhancedService.SendScheduledNotification(notification, scheduledAt); err != nil {
		h.logger.Error("Failed to schedule notification", "error", err)
		c.Error(errors.NewInternalError("Failed to schedule notification", err))
		return
	}

	h.logger.Info("Notification scheduled", "id", notification.ID, "scheduled_at", scheduledAt)
	c.JSON(http.StatusAccepted, gin.H{
		"message":         "Notification scheduled successfully",
		"notification_id": notification.ID,
		"scheduled_at":    scheduledAt,
	})
}

// GetMetrics returns notification dispatcher metrics
func (h *NotificationHandler) GetMetrics(c *gin.Context) {
	metrics := h.enhancedService.GetNotificationMetrics()
	c.JSON(http.StatusOK, gin.H{
		"metrics": metrics,
	})
}

// GetChannels returns available notification channels
func (h *NotificationHandler) GetChannels(c *gin.Context) {
	channels := h.enhancedService.GetAvailableChannels()
	c.JSON(http.StatusOK, gin.H{
		"channels": channels,
	})
}

// GetTemplates returns available notification templates
func (h *NotificationHandler) GetTemplates(c *gin.Context) {
	templates := h.enhancedService.GetTemplates()
	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
	})
}

// OrderConfirmationRequest represents order confirmation request
type OrderConfirmationRequest struct {
	CustomerID string                 `json:"customer_id" binding:"required"`
	OrderID    string                 `json:"order_id" binding:"required"`
	OrderData  map[string]interface{} `json:"order_data" binding:"required"`
}

// SendOrderConfirmation sends order confirmation notification
func (h *NotificationHandler) SendOrderConfirmation(c *gin.Context) {
	var req OrderConfirmationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if err := h.enhancedService.SendOrderConfirmation(req.CustomerID, req.OrderID, req.OrderData); err != nil {
		h.logger.Error("Failed to send order confirmation", "error", err)
		c.Error(errors.NewInternalError("Failed to send order confirmation", err))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Order confirmation sent successfully",
	})
}

// OrderShippedRequest represents order shipped request
type OrderShippedRequest struct {
	CustomerID     string `json:"customer_id" binding:"required"`
	OrderID        string `json:"order_id" binding:"required"`
	TrackingNumber string `json:"tracking_number" binding:"required"`
}

// SendOrderShipped sends order shipped notification
func (h *NotificationHandler) SendOrderShipped(c *gin.Context) {
	var req OrderShippedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if err := h.enhancedService.SendOrderShippedNotification(req.CustomerID, req.OrderID, req.TrackingNumber); err != nil {
		h.logger.Error("Failed to send order shipped notification", "error", err)
		c.Error(errors.NewInternalError("Failed to send order shipped notification", err))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Order shipped notification sent successfully",
	})
}

// PaymentSuccessRequest represents payment success request
type PaymentSuccessRequest struct {
	CustomerID  string                 `json:"customer_id" binding:"required"`
	PaymentData map[string]interface{} `json:"payment_data" binding:"required"`
}

// SendPaymentSuccess sends payment success notification
func (h *NotificationHandler) SendPaymentSuccess(c *gin.Context) {
	var req PaymentSuccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if err := h.enhancedService.SendPaymentSuccessNotification(req.CustomerID, req.PaymentData); err != nil {
		h.logger.Error("Failed to send payment success notification", "error", err)
		c.Error(errors.NewInternalError("Failed to send payment success notification", err))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Payment success notification sent successfully",
	})
}

// WelcomeRequest represents welcome notification request
type WelcomeRequest struct {
	CustomerID   string `json:"customer_id" binding:"required"`
	CustomerName string `json:"customer_name" binding:"required"`
}

// SendWelcome sends welcome notification
func (h *NotificationHandler) SendWelcome(c *gin.Context) {
	var req WelcomeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if err := h.enhancedService.SendWelcomeNotification(req.CustomerID, req.CustomerName); err != nil {
		h.logger.Error("Failed to send welcome notification", "error", err)
		c.Error(errors.NewInternalError("Failed to send welcome notification", err))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Welcome notification sent successfully",
	})
}

// LowStockAlertRequest represents low stock alert request
type LowStockAlertRequest struct {
	ProductName  string `json:"product_name" binding:"required"`
	CurrentStock int    `json:"current_stock" binding:"required"`
}

// SendLowStockAlert sends low stock alert
func (h *NotificationHandler) SendLowStockAlert(c *gin.Context) {
	var req LowStockAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid request"))
		return
	}

	if err := h.enhancedService.SendLowStockAlert(req.ProductName, req.CurrentStock); err != nil {
		h.logger.Error("Failed to send low stock alert", "error", err)
		c.Error(errors.NewInternalError("Failed to send low stock alert", err))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Low stock alert sent successfully",
	})
}
