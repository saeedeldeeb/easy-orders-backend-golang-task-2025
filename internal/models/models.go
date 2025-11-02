package models

import (
	"gorm.io/gorm"
)

// AllModels returns all model structs for migration
func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&Product{},
		&Inventory{},
		&Order{},
		&OrderItem{},
		&Payment{},
		&Notification{},
		&AuditLog{},
	}
}

// MigrateAll runs auto-migration for all models
func MigrateAll(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}

// CreateIndexes creates additional indexes for performance
func CreateIndexes(db *gorm.DB) error {
	// Composite indexes for better query performance

	// Orders: frequently queried by user and status
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_orders_user_status ON orders (user_id, status)").Error; err != nil {
		return err
	}

	// Orders: frequently queried by status and created_at for admin views
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_orders_status_created ON orders (status, created_at DESC)").Error; err != nil {
		return err
	}

	// Order Items: frequently queried by order and product
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_order_items_order_product ON order_items (order_id, product_id)").Error; err != nil {
		return err
	}

	// Payments: frequently queried by order and status
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payments_order_status ON payments (order_id, status)").Error; err != nil {
		return err
	}

	// Notifications: frequently queried by user and read status
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_notifications_user_read ON notifications (user_id, read, created_at DESC)").Error; err != nil {
		return err
	}

	// Audit Logs: frequently queried by entity
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs (entity_type, entity_id, created_at DESC)").Error; err != nil {
		return err
	}

	// Audit Logs: frequently queried by user and action
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_logs_user_action ON audit_logs (user_id, action, created_at DESC)").Error; err != nil {
		return err
	}

	// Products: search and filtering indexes
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_products_active_name ON products (is_active, name) WHERE deleted_at IS NULL").Error; err != nil {
		return err
	}

	// Inventory: low stock alerts
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_inventory_low_stock ON inventory (available, min_stock) WHERE available <= min_stock").Error; err != nil {
		return err
	}

	return nil
}

// CreateConstraints creates additional database constraints
func CreateConstraints(db *gorm.DB) error {
	// Ensure inventory quantities are non-negative
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_inventory_quantities'
			) THEN
				ALTER TABLE inventory ADD CONSTRAINT chk_inventory_quantities
				CHECK (quantity >= 0 AND reserved >= 0 AND available >= 0);
			END IF;
		END $$;
	`).Error; err != nil {
		return err
	}

	// Ensure reserved quantity doesn't exceed total quantity
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_inventory_reserved'
			) THEN
				ALTER TABLE inventory ADD CONSTRAINT chk_inventory_reserved
				CHECK (reserved <= quantity);
			END IF;
		END $$;
	`).Error; err != nil {
		return err
	}

	// Ensure available quantity is calculated correctly
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_inventory_available'
			) THEN
				ALTER TABLE inventory ADD CONSTRAINT chk_inventory_available
				CHECK (available = quantity - reserved);
			END IF;
		END $$;
	`).Error; err != nil {
		return err
	}

	// Ensure order item quantities are positive
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_order_items_quantity'
			) THEN
				ALTER TABLE order_items ADD CONSTRAINT chk_order_items_quantity
				CHECK (quantity > 0);
			END IF;
		END $$;
	`).Error; err != nil {
		return err
	}

	// Ensure order item prices are non-negative
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_order_items_prices'
			) THEN
				ALTER TABLE order_items ADD CONSTRAINT chk_order_items_prices
				CHECK (unit_price >= 0 AND total_price >= 0);
			END IF;
		END $$;
	`).Error; err != nil {
		return err
	}

	// Ensure payment amounts are positive
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_payments_amount'
			) THEN
				ALTER TABLE payments ADD CONSTRAINT chk_payments_amount
				CHECK (amount > 0);
			END IF;
		END $$;
	`).Error; err != nil {
		return err
	}

	// Ensure order total amounts are non-negative
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_orders_total'
			) THEN
				ALTER TABLE orders ADD CONSTRAINT chk_orders_total
				CHECK (total_amount >= 0);
			END IF;
		END $$;
	`).Error; err != nil {
		return err
	}

	return nil
}
