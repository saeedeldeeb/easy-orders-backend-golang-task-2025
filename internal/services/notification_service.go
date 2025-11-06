package services

import (
	"context"
	"errors"
	"fmt"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/logger"
)

// notificationService implements NotificationService interface
type notificationService struct {
	notificationRepo repository.NotificationRepository
	userRepo         repository.UserRepository
	logger           *logger.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	notificationRepo repository.NotificationRepository,
	userRepo repository.UserRepository,
	logger *logger.Logger,
) NotificationService {
	return &notificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		logger:           logger,
	}
}

func (s *notificationService) SendNotification(ctx context.Context, req SendNotificationRequest) error {
	s.logger.Info("Sending notification", "user_id", req.UserID, "type", req.Type, "channel", req.Channel)

	// Validate request
	if req.UserID == "" {
		return errors.New("user ID is required")
	}
	if req.Title == "" {
		return errors.New("notification title is required")
	}
	if req.Body == "" {
		return errors.New("notification body is required")
	}

	// Check if a user exists
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		s.logger.Error("Failed to get user for notification", "error", err, "user_id", req.UserID)
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Set defaults if not provided
	notificationType := models.NotificationType(req.Type)
	if notificationType == "" {
		notificationType = models.NotificationTypeSystem
	}

	notificationChannel := models.NotificationChannel(req.Channel)
	if notificationChannel == "" {
		notificationChannel = models.NotificationChannelInApp
	}

	// Create notification
	notification := &models.Notification{
		UserID:  req.UserID,
		Type:    notificationType,
		Channel: notificationChannel,
		Title:   req.Title,
		Body:    req.Body,
		Data:    req.Data,
		Read:    false,
	}

	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		s.logger.Error("Failed to create notification", "error", err, "user_id", req.UserID)
		return err
	}

	// Simulate sending notification via the specified channel
	if err := s.simulateSendNotification(ctx, notification); err != nil {
		s.logger.Error("Failed to send notification", "error", err, "notification_id", notification.ID)
		// Don't fail the request, just log the error
		return nil
	}

	// Mark the notification as sent
	notification.MarkAsSent()
	// Note: In a real implementation, we would need an Update method in the repository
	// For now, we'll just log that the notification was sent
	s.logger.Debug("Notification marked as sent", "notification_id", notification.ID)

	s.logger.Info("Notification sent successfully", "notification_id", notification.ID, "user_id", req.UserID)
	return nil
}

func (s *notificationService) GetUserNotifications(ctx context.Context, userID string, req ListNotificationsRequest) (*ListNotificationsResponse, error) {
	s.logger.Debug("Getting user notifications", "user_id", userID, "offset", req.Offset, "limit", req.Limit)

	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	// Check if a user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user for notifications", "error", err, "user_id", userID)
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Set a default limit if not provided
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20 // Default limit
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	var notifications []*models.Notification

	if req.UnreadOnly {
		notifications, err = s.notificationRepo.GetUnreadByUserID(ctx, userID, offset, limit)
	} else {
		notifications, err = s.notificationRepo.GetByUserID(ctx, userID, offset, limit)
	}

	if err != nil {
		s.logger.Error("Failed to get user notifications", "error", err, "user_id", userID)
		return nil, err
	}

	// Convert to response format
	notificationResponses := make([]*NotificationResponse, len(notifications))
	for i, notification := range notifications {
		notificationResponses[i] = &NotificationResponse{
			ID:      notification.ID,
			UserID:  notification.UserID,
			Type:    string(notification.Type),
			Channel: string(notification.Channel),
			Title:   notification.Title,
			Body:    notification.Body,
			Data:    notification.Data,
			Read:    notification.Read,
			ReadAt:  notification.ReadAt,
			SentAt:  notification.SentAt,
		}
	}

	s.logger.Debug("User notifications retrieved", "user_id", userID, "count", len(notificationResponses))

	return &ListNotificationsResponse{
		Notifications: notificationResponses,
		Offset:        offset,
		Limit:         limit,
		Total:         len(notificationResponses),
	}, nil
}

// simulateSendNotification simulates sending notification via different channels
// In a real implementation. This would integrate with email, SMS, push notification services
func (s *notificationService) simulateSendNotification(ctx context.Context, notification *models.Notification) error {
	s.logger.Debug("Simulating notification send", "notification_id", notification.ID, "channel", notification.Channel)

	switch notification.Channel {
	case models.NotificationChannelEmail:
		return s.simulateEmailNotification(ctx, notification)
	case models.NotificationChannelSMS:
		return s.simulateSMSNotification(ctx, notification)
	case models.NotificationChannelPush:
		return s.simulatePushNotification(ctx, notification)
	case models.NotificationChannelInApp:
		return s.simulateInAppNotification(ctx, notification)
	default:
		return fmt.Errorf("unsupported notification channel: %s", notification.Channel)
	}
}

func (s *notificationService) simulateEmailNotification(ctx context.Context, notification *models.Notification) error {
	s.logger.Debug("Simulating email notification", "notification_id", notification.ID)
	// Simulate email sending delay
	// In reality, this would integrate with email service like SendGrid, AWS SES, etc.
	return nil
}

func (s *notificationService) simulateSMSNotification(ctx context.Context, notification *models.Notification) error {
	s.logger.Debug("Simulating SMS notification", "notification_id", notification.ID)
	// Simulate SMS sending delay
	// In reality, this would integrate with SMS service like Twilio, AWS SNS, etc.
	return nil
}

func (s *notificationService) simulatePushNotification(ctx context.Context, notification *models.Notification) error {
	s.logger.Debug("Simulating push notification", "notification_id", notification.ID)
	// Simulate push notification delay
	// In reality, this would integrate with push service like Firebase Cloud Messaging, Apple Push Notification service, etc.
	return nil
}

func (s *notificationService) simulateInAppNotification(ctx context.Context, notification *models.Notification) error {
	s.logger.Debug("Simulating in-app notification", "notification_id", notification.ID)
	// In-app notifications are just stored in the database and shown in the UI.
	// No external service integration needed
	return nil
}
