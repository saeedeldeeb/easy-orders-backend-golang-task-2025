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

	// Check availability for all items before reserving any
	for _, item := range items {
		available, err := s.CheckAvailability(ctx, item.ProductID, item.Quantity)
		if err != nil {
			s.logger.Error("Failed to check availability during reservation", "error", err, "product_id", item.ProductID)
			return err
		}
		if !available {
			return fmt.Errorf("insufficient stock for product %s: requested %d", item.ProductID, item.Quantity)
		}
	}

	// Reserve inventory for each item
	for _, item := range items {
		if err := s.inventoryRepo.ReserveStock(ctx, item.ProductID, item.Quantity); err != nil {
			s.logger.Error("Failed to reserve stock", "error", err, "product_id", item.ProductID, "quantity", item.Quantity)

			// TODO: Rollback previously reserved items
			// For now, log the error and continue
			return fmt.Errorf("failed to reserve stock for product %s: %w", item.ProductID, err)
		}
		s.logger.Debug("Stock reserved", "product_id", item.ProductID, "quantity", item.Quantity)
	}

	s.logger.Info("Inventory reservation completed", "items_count", len(items))
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

	// Release inventory for each item
	for _, item := range items {
		if err := s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity); err != nil {
			s.logger.Error("Failed to release stock", "error", err, "product_id", item.ProductID, "quantity", item.Quantity)
			return fmt.Errorf("failed to release stock for product %s: %w", item.ProductID, err)
		}
		s.logger.Debug("Stock released", "product_id", item.ProductID, "quantity", item.Quantity)
	}

	s.logger.Info("Inventory release completed", "items_count", len(items))
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
