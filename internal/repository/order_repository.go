package repository

import (
	"context"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/logger"

	"gorm.io/gorm"
)

// orderRepository implements OrderRepository interface
type orderRepository struct {
	db     *database.DB
	logger *logger.Logger
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *database.DB, logger *logger.Logger) OrderRepository {
	return &orderRepository{
		db:     db,
		logger: logger,
	}
}

func (r *orderRepository) Create(ctx context.Context, order *models.Order) error {
	r.logger.Debug("Creating order in database", "user_id", order.UserID, "total", order.TotalAmount)

	if err := r.db.WithContext(ctx).Create(order).Error; err != nil {
		r.logger.Error("Failed to create order", "error", err, "user_id", order.UserID)
		return err
	}

	r.logger.Info("Order created in database", "id", order.ID, "user_id", order.UserID, "total", order.TotalAmount)
	return nil
}

func (r *orderRepository) GetByID(ctx context.Context, id string) (*models.Order, error) {
	r.logger.Debug("Getting order by ID", "id", id)

	var order models.Order
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Items").
		Preload("Items.Product").
		Preload("Payments").
		First(&order, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Debug("Order not found", "id", id)
			return nil, nil
		}
		r.logger.Error("Failed to get order by ID", "error", err, "id", id)
		return nil, err
	}

	r.logger.Debug("Order retrieved from database", "id", id, "status", order.Status)
	return &order, nil
}

func (r *orderRepository) GetByIDWithItems(ctx context.Context, id string) (*models.Order, error) {
	r.logger.Debug("Getting order with items by ID", "id", id)

	var order models.Order
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Items").
		Preload("Items.Product").
		Preload("Items.Product.Inventory").
		Preload("Payments").
		First(&order, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Debug("Order not found", "id", id)
			return nil, nil
		}
		r.logger.Error("Failed to get order with items by ID", "error", err, "id", id)
		return nil, err
	}

	r.logger.Debug("Order with items retrieved from database", "id", id, "items_count", len(order.Items))
	return &order, nil
}

func (r *orderRepository) GetByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.Order, error) {
	r.logger.Debug("Getting orders by user ID", "user_id", userID, "offset", offset, "limit", limit)

	var orders []*models.Order
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Items.Product").
		Preload("Payments").
		Where("user_id = ?", userID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		r.logger.Error("Failed to get orders by user ID", "error", err, "user_id", userID)
		return nil, err
	}

	r.logger.Debug("Orders retrieved for user", "user_id", userID, "count", len(orders))
	return orders, nil
}

func (r *orderRepository) Update(ctx context.Context, order *models.Order) error {
	r.logger.Debug("Updating order in database", "id", order.ID)

	if err := r.db.WithContext(ctx).Save(order).Error; err != nil {
		r.logger.Error("Failed to update order", "error", err, "id", order.ID)
		return err
	}

	r.logger.Info("Order updated in database", "id", order.ID, "status", order.Status)
	return nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id string, status models.OrderStatus) error {
	r.logger.Debug("Updating order status", "id", id, "status", status)

	result := r.db.WithContext(ctx).
		Model(&models.Order{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		r.logger.Error("Failed to update order status", "error", result.Error, "id", id)
		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("No order found to update status", "id", id)
		return gorm.ErrRecordNotFound
	}

	r.logger.Info("Order status updated", "id", id, "status", status)
	return nil
}

func (r *orderRepository) List(ctx context.Context, offset, limit int) ([]*models.Order, error) {
	r.logger.Debug("Listing orders from database", "offset", offset, "limit", limit)

	var orders []*models.Order
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Items").
		Preload("Items.Product").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		r.logger.Error("Failed to list orders", "error", err)
		return nil, err
	}

	r.logger.Debug("Orders retrieved from database", "count", len(orders))
	return orders, nil
}

func (r *orderRepository) ListByStatus(ctx context.Context, status models.OrderStatus, offset, limit int) ([]*models.Order, error) {
	r.logger.Debug("Listing orders by status", "status", status, "offset", offset, "limit", limit)

	var orders []*models.Order
	if err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Items").
		Preload("Items.Product").
		Where("status = ?", status).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		r.logger.Error("Failed to list orders by status", "error", err, "status", status)
		return nil, err
	}

	r.logger.Debug("Orders by status retrieved from database", "status", status, "count", len(orders))
	return orders, nil
}
