package repository

import (
	"context"
	"fmt"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/logger"

	"gorm.io/gorm"
)

// inventoryRepository implements InventoryRepository interface
type inventoryRepository struct {
	db     *database.DB
	logger *logger.Logger
}

// NewInventoryRepository creates a new inventory repository
func NewInventoryRepository(db *database.DB, logger *logger.Logger) InventoryRepository {
	return &inventoryRepository{
		db:     db,
		logger: logger,
	}
}

func (r *inventoryRepository) Create(ctx context.Context, inventory *models.Inventory) error {
	r.logger.Debug("Creating inventory", "product_id", inventory.ProductID, "quantity", inventory.Quantity)

	if err := r.db.WithContext(ctx).Create(inventory).Error; err != nil {
		r.logger.Error("Failed to create inventory", "error", err, "product_id", inventory.ProductID)
		return err
	}

	r.logger.Info("Inventory created successfully", "product_id", inventory.ProductID, "quantity", inventory.Quantity, "available", inventory.Available)
	return nil
}

func (r *inventoryRepository) GetByProductID(ctx context.Context, productID string) (*models.Inventory, error) {
	r.logger.Debug("Getting inventory by product ID", "product_id", productID)

	var inventory models.Inventory
	if err := r.db.WithContext(ctx).
		Preload("Product").
		First(&inventory, "product_id = ?", productID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Debug("Inventory not found", "product_id", productID)
			return nil, nil
		}
		r.logger.Error("Failed to get inventory by product ID", "error", err, "product_id", productID)
		return nil, err
	}

	r.logger.Debug("Inventory retrieved from database", "product_id", productID, "available", inventory.Available)
	return &inventory, nil
}

func (r *inventoryRepository) UpdateStock(ctx context.Context, productID string, quantity int) error {
	r.logger.Debug("Updating stock for product", "product_id", productID, "quantity", quantity)

	// Use optimistic locking to prevent race conditions
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inventory models.Inventory
		if err := tx.First(&inventory, "product_id = ?", productID).Error; err != nil {
			r.logger.Error("Failed to get inventory for update", "error", err, "product_id", productID)
			return err
		}

		oldVersion := inventory.Version
		inventory.Quantity = quantity
		inventory.Available = inventory.Quantity - inventory.Reserved
		inventory.Version++

		// Update with version check for optimistic locking
		result := tx.Model(&inventory).
			Where("product_id = ? AND version = ?", productID, oldVersion).
			Updates(map[string]interface{}{
				"quantity":  inventory.Quantity,
				"available": inventory.Available,
				"version":   inventory.Version,
			})

		if result.Error != nil {
			r.logger.Error("Failed to update inventory", "error", result.Error, "product_id", productID)
			return result.Error
		}

		if result.RowsAffected == 0 {
			r.logger.Warn("Inventory update failed due to version mismatch", "product_id", productID, "expected_version", oldVersion)
			return fmt.Errorf("inventory update conflict, please retry")
		}

		r.logger.Info("Inventory updated successfully", "product_id", productID, "new_quantity", quantity, "available", inventory.Available)
		return nil
	})
}

func (r *inventoryRepository) ReserveStock(ctx context.Context, productID string, quantity int) error {
	r.logger.Debug("Reserving stock for product", "product_id", productID, "quantity", quantity)

	// Use optimistic locking to prevent race conditions
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inventory models.Inventory
		if err := tx.First(&inventory, "product_id = ?", productID).Error; err != nil {
			r.logger.Error("Failed to get inventory for reservation", "error", err, "product_id", productID)
			return err
		}

		// Check if we can reserve the requested quantity
		if !inventory.CanReserve(quantity) {
			r.logger.Warn("Insufficient stock for reservation", "product_id", productID, "requested", quantity, "available", inventory.Available)
			return fmt.Errorf("insufficient stock: requested %d, available %d", quantity, inventory.Available)
		}

		oldVersion := inventory.Version
		if err := inventory.Reserve(quantity); err != nil {
			return err
		}
		inventory.Version++

		// Update with version check for optimistic locking
		result := tx.Model(&inventory).
			Where("product_id = ? AND version = ?", productID, oldVersion).
			Updates(map[string]interface{}{
				"reserved":  inventory.Reserved,
				"available": inventory.Available,
				"version":   inventory.Version,
			})

		if result.Error != nil {
			r.logger.Error("Failed to reserve inventory", "error", result.Error, "product_id", productID)
			return result.Error
		}

		if result.RowsAffected == 0 {
			r.logger.Warn("Inventory reservation failed due to version mismatch", "product_id", productID, "expected_version", oldVersion)
			return fmt.Errorf("inventory reservation conflict, please retry")
		}

		r.logger.Info("Inventory reserved successfully", "product_id", productID, "quantity", quantity, "reserved", inventory.Reserved, "available", inventory.Available)
		return nil
	})
}

func (r *inventoryRepository) ReleaseStock(ctx context.Context, productID string, quantity int) error {
	r.logger.Debug("Releasing stock for product", "product_id", productID, "quantity", quantity)

	// Use optimistic locking to prevent race conditions
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inventory models.Inventory
		if err := tx.First(&inventory, "product_id = ?", productID).Error; err != nil {
			r.logger.Error("Failed to get inventory for release", "error", err, "product_id", productID)
			return err
		}

		oldVersion := inventory.Version
		if err := inventory.Release(quantity); err != nil {
			r.logger.Error("Failed to release inventory", "error", err, "product_id", productID, "quantity", quantity)
			return err
		}
		inventory.Version++

		// Update with version check for optimistic locking
		result := tx.Model(&inventory).
			Where("product_id = ? AND version = ?", productID, oldVersion).
			Updates(map[string]interface{}{
				"reserved":  inventory.Reserved,
				"available": inventory.Available,
				"version":   inventory.Version,
			})

		if result.Error != nil {
			r.logger.Error("Failed to release inventory", "error", result.Error, "product_id", productID)
			return result.Error
		}

		if result.RowsAffected == 0 {
			r.logger.Warn("Inventory release failed due to version mismatch", "product_id", productID, "expected_version", oldVersion)
			return fmt.Errorf("inventory release conflict, please retry")
		}

		r.logger.Info("Inventory released successfully", "product_id", productID, "quantity", quantity, "reserved", inventory.Reserved, "available", inventory.Available)
		return nil
	})
}

func (r *inventoryRepository) FulfillStock(ctx context.Context, productID string, quantity int) error {
	r.logger.Debug("Fulfilling stock for product", "product_id", productID, "quantity", quantity)

	// Use optimistic locking to prevent race conditions
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inventory models.Inventory
		if err := tx.First(&inventory, "product_id = ?", productID).Error; err != nil {
			r.logger.Error("Failed to get inventory for fulfillment", "error", err, "product_id", productID)
			return err
		}

		oldVersion := inventory.Version
		if err := inventory.Fulfill(quantity); err != nil {
			r.logger.Error("Failed to fulfill inventory", "error", err, "product_id", productID, "quantity", quantity)
			return err
		}
		inventory.Version++

		// Update with version check for optimistic locking
		result := tx.Model(&inventory).
			Where("product_id = ? AND version = ?", productID, oldVersion).
			Updates(map[string]interface{}{
				"quantity":  inventory.Quantity,
				"reserved":  inventory.Reserved,
				"available": inventory.Available,
				"version":   inventory.Version,
			})

		if result.Error != nil {
			r.logger.Error("Failed to fulfill inventory", "error", result.Error, "product_id", productID)
			return result.Error
		}

		if result.RowsAffected == 0 {
			r.logger.Warn("Inventory fulfillment failed due to version mismatch", "product_id", productID, "expected_version", oldVersion)
			return fmt.Errorf("inventory fulfillment conflict, please retry")
		}

		r.logger.Info("Inventory fulfilled successfully", "product_id", productID, "quantity", quantity, "total_quantity", inventory.Quantity, "reserved", inventory.Reserved, "available", inventory.Available)
		return nil
	})
}

func (r *inventoryRepository) GetLowStockItems(ctx context.Context, threshold int) ([]*models.Inventory, error) {
	r.logger.Debug("Getting low stock items", "threshold", threshold)

	var inventories []*models.Inventory
	if err := r.db.WithContext(ctx).
		Preload("Product").
		Where("available <= ?", threshold).
		Order("available ASC").
		Find(&inventories).Error; err != nil {
		r.logger.Error("Failed to get low stock items", "error", err)
		return nil, err
	}

	r.logger.Debug("Low stock items retrieved", "count", len(inventories), "threshold", threshold)
	return inventories, nil
}

func (r *inventoryRepository) BulkReserve(ctx context.Context, items []InventoryReservation) error {
	r.logger.Debug("Bulk reserving inventory items", "count", len(items))

	// Use a single transaction for all operations
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Track successful reservations for rollback
		var reservedItems []InventoryReservation

		for _, item := range items {
			// Reserve using the transaction context
			var inventory models.Inventory
			if err := tx.First(&inventory, "product_id = ?", item.ProductID).Error; err != nil {
				r.logger.Error("Failed to get inventory for bulk reservation", "error", err, "product_id", item.ProductID)
				return err
			}

			// Check if we can reserve the requested quantity
			if !inventory.CanReserve(item.Quantity) {
				r.logger.Warn("Insufficient stock for bulk reservation",
					"product_id", item.ProductID,
					"requested", item.Quantity,
					"available", inventory.Available)
				return fmt.Errorf("insufficient stock for product %s: requested %d, available %d",
					item.ProductID, item.Quantity, inventory.Available)
			}

			oldVersion := inventory.Version
			if err := inventory.Reserve(item.Quantity); err != nil {
				return err
			}
			inventory.Version++

			// Update with version check for optimistic locking
			result := tx.Model(&inventory).
				Where("product_id = ? AND version = ?", item.ProductID, oldVersion).
				Updates(map[string]interface{}{
					"reserved":  inventory.Reserved,
					"available": inventory.Available,
					"version":   inventory.Version,
				})

			if result.Error != nil {
				r.logger.Error("Failed to reserve inventory in bulk", "error", result.Error, "product_id", item.ProductID)
				return result.Error
			}

			if result.RowsAffected == 0 {
				r.logger.Warn("Bulk inventory reservation failed due to version mismatch",
					"product_id", item.ProductID, "expected_version", oldVersion)
				return fmt.Errorf("inventory reservation conflict for product %s, please retry", item.ProductID)
			}

			reservedItems = append(reservedItems, item)
			r.logger.Debug("Item reserved in bulk operation", "product_id", item.ProductID, "quantity", item.Quantity)
		}

		r.logger.Info("Bulk inventory reservation completed successfully", "count", len(reservedItems))
		return nil
	})
}

func (r *inventoryRepository) BulkRelease(ctx context.Context, items []InventoryReservation) error {
	r.logger.Debug("Bulk releasing inventory items", "count", len(items))

	// Use a single transaction for all operations
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var releasedItems []InventoryReservation

		for _, item := range items {
			// Release using the transaction context
			var inventory models.Inventory
			if err := tx.First(&inventory, "product_id = ?", item.ProductID).Error; err != nil {
				r.logger.Error("Failed to get inventory for bulk release", "error", err, "product_id", item.ProductID)
				return err
			}

			oldVersion := inventory.Version
			if err := inventory.Release(item.Quantity); err != nil {
				r.logger.Error("Failed to release inventory in bulk", "error", err, "product_id", item.ProductID, "quantity", item.Quantity)
				return err
			}
			inventory.Version++

			// Update with version check for optimistic locking
			result := tx.Model(&inventory).
				Where("product_id = ? AND version = ?", item.ProductID, oldVersion).
				Updates(map[string]interface{}{
					"reserved":  inventory.Reserved,
					"available": inventory.Available,
					"version":   inventory.Version,
				})

			if result.Error != nil {
				r.logger.Error("Failed to release inventory in bulk", "error", result.Error, "product_id", item.ProductID)
				return result.Error
			}

			if result.RowsAffected == 0 {
				r.logger.Warn("Bulk inventory release failed due to version mismatch",
					"product_id", item.ProductID, "expected_version", oldVersion)
				return fmt.Errorf("inventory release conflict for product %s, please retry", item.ProductID)
			}

			releasedItems = append(releasedItems, item)
			r.logger.Debug("Item released in bulk operation", "product_id", item.ProductID, "quantity", item.Quantity)
		}

		r.logger.Info("Bulk inventory release completed successfully", "count", len(releasedItems))
		return nil
	})
}
