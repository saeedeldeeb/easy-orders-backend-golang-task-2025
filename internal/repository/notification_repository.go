package repository

import (
	"context"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/logger"

	"gorm.io/gorm"
)

// notificationRepository implements NotificationRepository interface
type notificationRepository struct {
	db     *database.DB
	logger *logger.Logger
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *database.DB, logger *logger.Logger) NotificationRepository {
	return &notificationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *notificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	r.logger.Debug("Creating notification in database", "user_id", notification.UserID, "type", notification.Type, "channel", notification.Channel)

	if err := r.db.WithContext(ctx).Create(notification).Error; err != nil {
		r.logger.Error("Failed to create notification", "error", err, "user_id", notification.UserID)
		return err
	}

	r.logger.Info("Notification created in database", "id", notification.ID, "user_id", notification.UserID, "type", notification.Type)
	return nil
}

func (r *notificationRepository) GetByID(ctx context.Context, id string) (*models.Notification, error) {
	r.logger.Debug("Getting notification by ID", "id", id)

	var notification models.Notification
	if err := r.db.WithContext(ctx).
		Preload("User").
		First(&notification, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Debug("Notification not found", "id", id)
			return nil, nil
		}
		r.logger.Error("Failed to get notification by ID", "error", err, "id", id)
		return nil, err
	}

	r.logger.Debug("Notification retrieved from database", "id", id, "read", notification.Read)
	return &notification, nil
}

func (r *notificationRepository) GetByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.Notification, error) {
	r.logger.Debug("Getting notifications by user ID", "user_id", userID, "offset", offset, "limit", limit)

	var notifications []*models.Notification
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		r.logger.Error("Failed to get notifications by user ID", "error", err, "user_id", userID)
		return nil, err
	}

	r.logger.Debug("Notifications retrieved for user", "user_id", userID, "count", len(notifications))
	return notifications, nil
}

func (r *notificationRepository) GetUnreadByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.Notification, error) {
	r.logger.Debug("Getting unread notifications by user ID", "user_id", userID, "offset", offset, "limit", limit)

	var notifications []*models.Notification
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND read = ?", userID, false).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		r.logger.Error("Failed to get unread notifications by user ID", "error", err, "user_id", userID)
		return nil, err
	}

	r.logger.Debug("Unread notifications retrieved for user", "user_id", userID, "count", len(notifications))
	return notifications, nil
}
