package services

import (
	"context"
	"errors"
	"fmt"

	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/logger"
)

// inventoryService implements InventoryService interface
type inventoryService struct {
	inventoryRepo repository.InventoryRepository
	productRepo   repository.ProductRepository
	logger        *logger.Logger
}

// NewInventoryService creates a new inventory service
func NewInventoryService(
	inventoryRepo repository.InventoryRepository,
	productRepo repository.ProductRepository,
	logger *logger.Logger,
) InventoryService {
	return &inventoryService{
		inventoryRepo: inventoryRepo,
		productRepo:   productRepo,
		logger:        logger,
	}
}

func (s *inventoryService) CheckAvailability(ctx context.Context, productID string, quantity int) (bool, error) {
	s.logger.Debug("Checking inventory availability", "product_id", productID, "quantity", quantity)

	if productID == "" {
		return false, errors.New("product ID is required")
	}
	if quantity <= 0 {
		return false, errors.New("quantity must be greater than 0")
	}

	// Get inventory for the product
	inventory, err := s.inventoryRepo.GetByProductID(ctx, productID)
	if err != nil {
		s.logger.Error("Failed to get inventory", "error", err, "product_id", productID)
		return false, err
	}

	if inventory == nil {
		s.logger.Debug("No inventory found for product", "product_id", productID)
		return false, nil
	}

	available := inventory.CanReserve(quantity)
	s.logger.Debug("Inventory availability checked", "product_id", productID, "requested", quantity, "available", inventory.Available, "can_reserve", available)

	return available, nil
}

func (s *inventoryService) ReserveInventory(ctx context.Context, items []InventoryItem) error {
	s.logger.Debug("Reserving inventory", "items_count", len(items))

	if len(items) == 0 {
		return errors.New("no items to reserve")
	}

	// Validate all items first
	for _, item := range items {
		if item.ProductID == "" {
			return errors.New("product ID is required for all items")
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("quantity must be greater than 0 for product %s", item.ProductID)
		}
	}

	// Convert to repository reservation format
	reservations := make([]repository.InventoryReservation, len(items))
	for i, item := range items {
		reservations[i] = repository.InventoryReservation{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	// Use bulk reserve which handles transactions and automatic rollback
	// If any item fails, all reservations are rolled back automatically
	if err := s.inventoryRepo.BulkReserve(ctx, reservations); err != nil {
		s.logger.Error("Failed to bulk reserve inventory", "error", err, "items_count", len(items))
		return fmt.Errorf("failed to reserve inventory: %w", err)
	}

	s.logger.Info("Inventory reservation completed successfully", "items_count", len(items))
	return nil
}

func (s *inventoryService) ReleaseInventory(ctx context.Context, items []InventoryItem) error {
	s.logger.Debug("Releasing inventory", "items_count", len(items))

	if len(items) == 0 {
		return errors.New("no items to release")
	}

	// Validate all items first
	for _, item := range items {
		if item.ProductID == "" {
			return errors.New("product ID is required for all items")
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("quantity must be greater than 0 for product %s", item.ProductID)
		}
	}

	// Convert to repository reservation format
	reservations := make([]repository.InventoryReservation, len(items))
	for i, item := range items {
		reservations[i] = repository.InventoryReservation{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	// Use bulk release which handles transactions and automatic rollback
	// If any item fails, all releases are rolled back automatically
	if err := s.inventoryRepo.BulkRelease(ctx, reservations); err != nil {
		s.logger.Error("Failed to bulk release inventory", "error", err, "items_count", len(items))
		return fmt.Errorf("failed to release inventory: %w", err)
	}

	s.logger.Info("Inventory release completed successfully", "items_count", len(items))
	return nil
}

func (s *inventoryService) GetLowStockAlert(ctx context.Context, threshold int) (*LowStockResponse, error) {
	s.logger.Debug("Getting low stock alert", "threshold", threshold)

	if threshold < 0 {
		threshold = 10 // Default threshold
	}

	lowStockItems, err := s.inventoryRepo.GetLowStockItems(ctx, threshold)
	if err != nil {
		s.logger.Error("Failed to get low stock items", "error", err, "threshold", threshold)
		return nil, err
	}

	// Convert to response format
	alerts := make([]LowStockItem, len(lowStockItems))
	for i, inventory := range lowStockItems {
		productName := "Unknown Product"
		productSKU := ""
		if inventory.Product != nil {
			productName = inventory.Product.Name
			productSKU = inventory.Product.SKU
		}

		alerts[i] = LowStockItem{
			ProductID:    inventory.ProductID,
			ProductName:  productName,
			ProductSKU:   productSKU,
			CurrentStock: inventory.Available,
			MinStock:     inventory.MinStock,
		}
	}

	s.logger.Debug("Low stock alert generated", "alert_count", len(alerts), "threshold", threshold)

	return &LowStockResponse{
		Threshold: threshold,
		Products:  convertToProductLowStock(alerts),
		Count:     len(alerts),
	}, nil
}

// Helper function to convert LowStockItem to ProductLowStock
func convertToProductLowStock(items []LowStockItem) []ProductLowStock {
	products := make([]ProductLowStock, len(items))
	for i, item := range items {
		products[i] = ProductLowStock{
			ProductID:    item.ProductID,
			ProductName:  item.ProductName,
			SKU:          item.ProductSKU,
			CurrentStock: item.CurrentStock,
			MinThreshold: item.MinStock,
		}
	}
	return products
}
