package database

import (
	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/logger"

	"gorm.io/gorm"
)

// Migrator handles database migrations
type Migrator struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewMigrator creates a new database migrator
func NewMigrator(db *gorm.DB, logger *logger.Logger) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger,
	}
}

// RunMigrations runs all database migrations
func (m *Migrator) RunMigrations() error {
	m.logger.Info("Starting database migrations...")

	// Enable UUID extension for PostgreSQL
	if err := m.enableUUIDExtension(); err != nil {
		m.logger.Error("Failed to enable UUID extension", "error", err)
		return err
	}

	// Run auto-migrations for all models
	if err := models.MigrateAll(m.db); err != nil {
		m.logger.Error("Failed to run auto-migrations", "error", err)
		return err
	}

	// Create additional indexes
	if err := models.CreateIndexes(m.db); err != nil {
		m.logger.Error("Failed to create indexes", "error", err)
		return err
	}

	// Create database constraints
	if err := models.CreateConstraints(m.db); err != nil {
		m.logger.Error("Failed to create constraints", "error", err)
		return err
	}

	m.logger.Info("Database migrations completed successfully")
	return nil
}

// enableUUIDExtension enables the UUID extension for PostgreSQL
func (m *Migrator) enableUUIDExtension() error {
	m.logger.Debug("Enabling UUID extension...")

	// Enable uuid-ossp extension for UUID generation
	if err := m.db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return err
	}

	// Enable pgcrypto extension for additional UUID functions
	if err := m.db.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\"").Error; err != nil {
		return err
	}

	m.logger.Debug("UUID extensions enabled successfully")
	return nil
}

// RollbackMigrations rolls back migrations (for development/testing)
func (m *Migrator) RollbackMigrations() error {
	m.logger.Warn("Rolling back database migrations...")

	// Drop tables in reverse dependency order
	tables := []string{
		"audit_logs",
		"notifications",
		"payments",
		"order_items",
		"orders",
		"inventory",
		"products",
		"users",
	}

	for _, table := range tables {
		if err := m.db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE").Error; err != nil {
			m.logger.Error("Failed to drop table", "table", table, "error", err)
			return err
		}
		m.logger.Debug("Dropped table", "table", table)
	}

	m.logger.Warn("Database migrations rolled back")
	return nil
}

// SeedData seeds the database with initial data
func (m *Migrator) SeedData() error {
	m.logger.Info("Seeding database with initial data...")

	// Create admin user
	adminUser := &models.User{
		Email:    "admin@easy-orders.com",
		Password: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
		Name:     "System Administrator",
		Role:     models.UserRoleAdmin,
		IsActive: true,
	}

	if err := m.db.FirstOrCreate(adminUser, "email = ?", adminUser.Email).Error; err != nil {
		m.logger.Error("Failed to create admin user", "error", err)
		return err
	}

	// Create sample customer
	customerUser := &models.User{
		Email:    "customer@example.com",
		Password: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
		Name:     "John Doe",
		Role:     models.UserRoleCustomer,
		IsActive: true,
	}

	if err := m.db.FirstOrCreate(customerUser, "email = ?", customerUser.Email).Error; err != nil {
		m.logger.Error("Failed to create customer user", "error", err)
		return err
	}

	// Create sample products
	products := []*models.Product{
		{
			Name:        "Wireless Bluetooth Headphones",
			Description: "High-quality wireless headphones with noise cancellation",
			Price:       99.99,
			SKU:         "WBH-001",
			IsActive:    true,
		},
		{
			Name:        "Smartphone Case",
			Description: "Protective case for smartphones with multiple color options",
			Price:       24.99,
			SKU:         "SC-001",
			IsActive:    true,
		},
		{
			Name:        "USB-C Cable",
			Description: "Fast charging USB-C cable, 2 meters length",
			Price:       12.99,
			SKU:         "USBC-001",
			IsActive:    true,
		},
	}

	for _, product := range products {
		if err := m.db.FirstOrCreate(product, "sku = ?", product.SKU).Error; err != nil {
			m.logger.Error("Failed to create product", "sku", product.SKU, "error", err)
			return err
		}

		// Create inventory for each product
		inventory := &models.Inventory{
			ProductID: product.ID,
			Quantity:  100,
			Reserved:  0,
			Available: 100,
			MinStock:  10,
			MaxStock:  500,
		}

		if err := m.db.FirstOrCreate(inventory, "product_id = ?", product.ID).Error; err != nil {
			m.logger.Error("Failed to create inventory", "product_id", product.ID, "error", err)
			return err
		}
	}

	m.logger.Info("Database seeded successfully")
	return nil
}
