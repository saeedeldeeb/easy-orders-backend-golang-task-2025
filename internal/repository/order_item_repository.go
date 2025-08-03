package repository

import (
	"context"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/logger"

	"gorm.io/gorm"
)

// orderItemRepository implements OrderItemRepository interface
type orderItemRepository struct {
	db     *database.DB
	logger *logger.Logger
}

// NewOrderItemRepository creates a new order item repository
func NewOrderItemRepository(db *database.DB, logger *logger.Logger) OrderItemRepository {
	return &orderItemRepository{
		db:     db,
		logger: logger,
	}
}

func (r *orderItemRepository) Create(ctx context.Context, item *models.OrderItem) error {
	r.logger.Debug("Creating order item in database", "order_id", item.OrderID, "product_id", item.ProductID, "quantity", item.Quantity)

	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		r.logger.Error("Failed to create order item", "error", err, "order_id", item.OrderID, "product_id", item.ProductID)
		return err
	}

	r.logger.Info("Order item created in database", "id", item.ID, "order_id", item.OrderID, "product_id", item.ProductID)
	return nil
}

func (r *orderItemRepository) CreateBatch(ctx context.Context, items []*models.OrderItem) error {
	r.logger.Debug("Creating batch of order items", "count", len(items))

	if len(items) == 0 {
		r.logger.Debug("No items to create")
		return nil
	}

	// Use a transaction for batch creation
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Create(item).Error; err != nil {
				r.logger.Error("Failed to create order item in batch", "error", err, "order_id", item.OrderID, "product_id", item.ProductID)
				return err
			}
		}

		r.logger.Info("Batch order items created successfully", "count", len(items), "order_id", items[0].OrderID)
		return nil
	})
}

func (r *orderItemRepository) GetByOrderID(ctx context.Context, orderID string) ([]*models.OrderItem, error) {
	r.logger.Debug("Getting order items by order ID", "order_id", orderID)

	var items []*models.OrderItem
	if err := r.db.WithContext(ctx).
		Preload("Product").
		Preload("Product.Inventory").
		Where("order_id = ?", orderID).
		Order("created_at ASC").
		Find(&items).Error; err != nil {
		r.logger.Error("Failed to get order items by order ID", "error", err, "order_id", orderID)
		return nil, err
	}

	r.logger.Debug("Order items retrieved from database", "order_id", orderID, "count", len(items))
	return items, nil
}

func (r *orderItemRepository) Update(ctx context.Context, item *models.OrderItem) error {
	r.logger.Debug("Updating order item in database", "id", item.ID)

	if err := r.db.WithContext(ctx).Save(item).Error; err != nil {
		r.logger.Error("Failed to update order item", "error", err, "id", item.ID)
		return err
	}

	r.logger.Info("Order item updated in database", "id", item.ID)
	return nil
}

func (r *orderItemRepository) Delete(ctx context.Context, id string) error {
	r.logger.Debug("Deleting order item from database", "id", id)

	if err := r.db.WithContext(ctx).Delete(&models.OrderItem{}, "id = ?", id).Error; err != nil {
		r.logger.Error("Failed to delete order item", "error", err, "id", id)
		return err
	}

	r.logger.Info("Order item deleted from database", "id", id)
	return nil
}
