package services

import (
	"context"
	"fmt"
	"time"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/logger"
	"easy-orders-backend/pkg/notifications"
)

// EnhancedNotificationService provides advanced notification capabilities
type EnhancedNotificationService struct {
	notificationRepo repository.NotificationRepository
	dispatcher       *notifications.NotificationDispatcher
	provider         *notifications.NotificationProvider
	templateManager  *notifications.TemplateManager
	logger           *logger.Logger
}

// NewEnhancedNotificationService creates a new enhanced notification service
func NewEnhancedNotificationService(
	notificationRepo repository.NotificationRepository,
	dispatcher *notifications.NotificationDispatcher,
	provider *notifications.NotificationProvider,
	templateManager *notifications.TemplateManager,
	logger *logger.Logger,
) *EnhancedNotificationService {
	return &EnhancedNotificationService{
		notificationRepo: notificationRepo,
		dispatcher:       dispatcher,
		provider:         provider,
		templateManager:  templateManager,
		logger:           logger,
	}
}

// Start initializes the enhanced notification service
func (ens *EnhancedNotificationService) Start(ctx context.Context) error {
	// Initialize notification channels
	ens.initializeChannels()

	// Start the dispatcher
	if err := ens.dispatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start notification dispatcher: %w", err)
	}

	ens.logger.Info("Enhanced notification service started")
	return nil
}

// Stop gracefully shuts down the notification service
func (ens *EnhancedNotificationService) Stop() error {
	return ens.dispatcher.Stop()
}

// SendNotification sends a single notification (implements existing interface)
func (ens *EnhancedNotificationService) SendNotification(ctx context.Context, req SendNotificationRequest) error {
	notification := notifications.NewNotification(
		notifications.NotificationType(req.Type),
		"email", // default channel
		req.UserID,
	)

	notification.Body = req.Data
	notification.SetPriority(notifications.PriorityNormal)

	return ens.DispatchNotification(notification)
}

// DispatchNotification dispatches a notification asynchronously
func (ens *EnhancedNotificationService) DispatchNotification(notification *notifications.Notification) error {
	// Save to database first
	dbNotification := &models.Notification{
		UserID:    notification.Recipient,
		Type:      models.NotificationType(notification.Type),
		Channel:   models.NotificationChannel(notification.Channel),
		Title:     notification.Subject,
		Body:      notification.Body,
		Data:      notification.Body,
		CreatedAt: notification.CreatedAt,
	}

	if err := ens.notificationRepo.Create(context.Background(), dbNotification); err != nil {
		ens.logger.Error("Failed to save notification to database", "error", err)
		// Continue with dispatch even if DB save fails
	} else {
		notification.ID = dbNotification.ID
	}

	// Dispatch asynchronously
	return ens.dispatcher.Dispatch(notification)
}

// SendOrderConfirmation sends order confirmation notification
func (ens *EnhancedNotificationService) SendOrderConfirmation(customerID, orderID string, orderData map[string]interface{}) error {
	notification := notifications.NewNotification(
		notifications.NotificationTypeOrderConfirmation,
		"email",
		customerID,
	)

	notification.TemplateID = "order_confirmation_email"
	notification.Data = orderData
	notification.SetPriority(notifications.PriorityHigh)

	return ens.DispatchNotification(notification)
}

// SendOrderShippedNotification sends order shipped notification
func (ens *EnhancedNotificationService) SendOrderShippedNotification(customerID, orderID, trackingNumber string) error {
	notification := notifications.NewNotification(
		notifications.NotificationTypeOrderShipped,
		"email",
		customerID,
	)

	notification.TemplateID = "order_shipped_email"
	notification.Data = map[string]interface{}{
		"OrderID":        orderID,
		"TrackingNumber": trackingNumber,
	}
	notification.SetPriority(notifications.PriorityHigh)

	return ens.DispatchNotification(notification)
}

// SendPaymentSuccessNotification sends payment success notification
func (ens *EnhancedNotificationService) SendPaymentSuccessNotification(customerID string, paymentData map[string]interface{}) error {
	notification := notifications.NewNotification(
		notifications.NotificationTypePaymentSuccess,
		"email",
		customerID,
	)

	notification.TemplateID = "payment_success_email"
	notification.Data = paymentData
	notification.SetPriority(notifications.PriorityHigh)

	return ens.DispatchNotification(notification)
}

// SendWelcomeNotification sends welcome notification to new users
func (ens *EnhancedNotificationService) SendWelcomeNotification(customerID, customerName string) error {
	notification := notifications.NewNotification(
		notifications.NotificationTypeWelcome,
		"email",
		customerID,
	)

	notification.TemplateID = "welcome_email"
	notification.Data = map[string]interface{}{
		"CustomerName": customerName,
	}
	notification.SetPriority(notifications.PriorityNormal)

	return ens.DispatchNotification(notification)
}

// SendLowStockAlert sends low stock alert to admin
func (ens *EnhancedNotificationService) SendLowStockAlert(productName string, currentStock int) error {
	notification := notifications.NewNotification(
		notifications.NotificationTypeLowStock,
		"sms",
		"admin", // Send to admin
	)

	notification.TemplateID = "low_stock_sms"
	notification.Data = map[string]interface{}{
		"ProductName":  productName,
		"CurrentStock": currentStock,
	}
	notification.SetPriority(notifications.PriorityCritical)

	return ens.DispatchNotification(notification)
}

// SendBulkNotifications sends multiple notifications in batch
func (ens *EnhancedNotificationService) SendBulkNotifications(notifications []*notifications.Notification) error {
	// Save all to database first
	for _, notification := range notifications {
		dbNotification := &models.Notification{
			UserID:    notification.Recipient,
			Type:      models.NotificationType(notification.Type),
			Channel:   models.NotificationChannel(notification.Channel),
			Title:     notification.Subject,
			Body:      notification.Body,
			Data:      notification.Body,
			CreatedAt: notification.CreatedAt,
		}

		if err := ens.notificationRepo.Create(context.Background(), dbNotification); err != nil {
			ens.logger.Error("Failed to save bulk notification to database", "error", err)
		} else {
			notification.ID = dbNotification.ID
		}
	}

	// Dispatch in batch
	return ens.dispatcher.DispatchBatch(notifications)
}

// SendScheduledNotification schedules a notification for future delivery
func (ens *EnhancedNotificationService) SendScheduledNotification(notification *notifications.Notification, scheduledAt time.Time) error {
	notification.ScheduledAt = &scheduledAt
	return ens.DispatchNotification(notification)
}

// GetNotificationMetrics returns dispatcher metrics
func (ens *EnhancedNotificationService) GetNotificationMetrics() *notifications.DispatcherMetrics {
	return ens.dispatcher.GetMetrics()
}

// GetAvailableChannels returns available notification channels
func (ens *EnhancedNotificationService) GetAvailableChannels() []string {
	channels := ens.provider.GetAvailableChannels()
	names := make([]string, len(channels))
	for i, channel := range channels {
		names[i] = channel.GetName()
	}
	return names
}

// GetTemplates returns available notification templates
func (ens *EnhancedNotificationService) GetTemplates() []*notifications.Template {
	return ens.templateManager.ListTemplates()
}

// initializeChannels sets up notification channels
func (ens *EnhancedNotificationService) initializeChannels() {
	// Initialize email channel
	emailConfig := notifications.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "notifications@example.com",
		Password: "password",
		UseTLS:   true,
	}
	emailChannel := notifications.NewEmailChannel(emailConfig, ens.logger)
	ens.provider.RegisterChannel(emailChannel)

	// Initialize SMS channel
	smsChannel := notifications.NewSMSChannel("sms-api-key", "https://api.sms.com", ens.logger)
	ens.provider.RegisterChannel(smsChannel)

	// Initialize push notification channel
	pushChannel := notifications.NewPushChannel(ens.logger)
	ens.provider.RegisterChannel(pushChannel)

	// Initialize webhook channel
	webhookChannel := notifications.NewWebhookChannel("https://webhook.example.com", ens.logger)
	ens.provider.RegisterChannel(webhookChannel)

	ens.logger.Info("Notification channels initialized", "count", 4)
}

// Base NotificationService interface implementation (backward compatibility)
func (ens *EnhancedNotificationService) ListNotifications(ctx context.Context, req ListNotificationsRequest) (*ListNotificationsResponse, error) {
	// For now, we'll use a dummy user ID - in a real implementation this would come from the request context
	userID := "current-user" // This should be extracted from authentication context

	var notifications []*models.Notification
	var err error

	if req.UnreadOnly {
		notifications, err = ens.notificationRepo.GetUnreadByUserID(ctx, userID, req.Offset, req.Limit)
	} else {
		notifications, err = ens.notificationRepo.GetByUserID(ctx, userID, req.Offset, req.Limit)
	}

	if err != nil {
		return nil, err
	}

	responses := make([]*NotificationResponse, len(notifications))
	for i, notif := range notifications {
		responses[i] = &NotificationResponse{
			ID:      notif.ID,
			UserID:  notif.UserID,
			Type:    string(notif.Type),
			Channel: string(notif.Channel),
			Data:    notif.Data,
			ReadAt:  notif.ReadAt,
		}
	}

	return &ListNotificationsResponse{
		Notifications: responses,
		Offset:        req.Offset,
		Limit:         req.Limit,
		Total:         len(notifications), // simplified for now
	}, nil
}

func (ens *EnhancedNotificationService) MarkAsRead(ctx context.Context, notificationID string) error {
	return ens.notificationRepo.MarkAsRead(ctx, notificationID)
}

func (ens *EnhancedNotificationService) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	return ens.notificationRepo.GetUnreadCount(ctx, userID)
}
