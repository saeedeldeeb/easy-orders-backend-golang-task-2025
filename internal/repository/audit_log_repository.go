package repository

import (
	"context"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/logger"

	"gorm.io/gorm"
)

// auditLogRepository implements AuditLogRepository interface
type auditLogRepository struct {
	db     *database.DB
	logger *logger.Logger
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *database.DB, logger *logger.Logger) AuditLogRepository {
	return &auditLogRepository{
		db:     db,
		logger: logger,
	}
}

func (r *auditLogRepository) Create(ctx context.Context, log *models.AuditLog) error {
	r.logger.Debug("Creating audit log in database", "entity_type", log.EntityType, "entity_id", log.EntityID, "action", log.Action)

	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		r.logger.Error("Failed to create audit log", "error", err, "entity_type", log.EntityType, "entity_id", log.EntityID)
		return err
	}

	r.logger.Debug("Audit log created in database", "id", log.ID, "entity_type", log.EntityType, "action", log.Action)
	return nil
}

func (r *auditLogRepository) GetByID(ctx context.Context, id string) (*models.AuditLog, error) {
	r.logger.Debug("Getting audit log by ID", "id", id)

	var auditLog models.AuditLog
	if err := r.db.WithContext(ctx).
		Preload("User").
		First(&auditLog, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Debug("Audit log not found", "id", id)
			return nil, nil
		}
		r.logger.Error("Failed to get audit log by ID", "error", err, "id", id)
		return nil, err
	}

	r.logger.Debug("Audit log retrieved from database", "id", id, "action", auditLog.Action)
	return &auditLog, nil
}

func (r *auditLogRepository) GetByEntityID(ctx context.Context, entityType, entityID string, offset, limit int) ([]*models.AuditLog, error) {
	r.logger.Debug("Getting audit logs by entity", "entity_type", entityType, "entity_id", entityID, "offset", offset, "limit", limit)

	var auditLogs []*models.AuditLog
	if err := r.db.WithContext(ctx).
		Preload("User").
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&auditLogs).Error; err != nil {
		r.logger.Error("Failed to get audit logs by entity", "error", err, "entity_type", entityType, "entity_id", entityID)
		return nil, err
	}

	r.logger.Debug("Audit logs retrieved for entity", "entity_type", entityType, "entity_id", entityID, "count", len(auditLogs))
	return auditLogs, nil
}

func (r *auditLogRepository) GetByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.AuditLog, error) {
	r.logger.Debug("Getting audit logs by user ID", "user_id", userID, "offset", offset, "limit", limit)

	var auditLogs []*models.AuditLog
	if err := r.db.WithContext(ctx).
		Preload("User").
		Where("user_id = ?", userID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&auditLogs).Error; err != nil {
		r.logger.Error("Failed to get audit logs by user ID", "error", err, "user_id", userID)
		return nil, err
	}

	r.logger.Debug("Audit logs retrieved for user", "user_id", userID, "count", len(auditLogs))
	return auditLogs, nil
}

func (r *auditLogRepository) List(ctx context.Context, offset, limit int) ([]*models.AuditLog, error) {
	r.logger.Debug("Listing audit logs from database", "offset", offset, "limit", limit)

	var auditLogs []*models.AuditLog
	if err := r.db.WithContext(ctx).
		Preload("User").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&auditLogs).Error; err != nil {
		r.logger.Error("Failed to list audit logs", "error", err)
		return nil, err
	}

	r.logger.Debug("Audit logs retrieved from database", "count", len(auditLogs))
	return auditLogs, nil
}
