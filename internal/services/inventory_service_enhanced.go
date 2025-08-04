package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/concurrency"
	"easy-orders-backend/pkg/logger"
)

// enhancedInventoryService provides advanced concurrent inventory management
type enhancedInventoryService struct {
	inventoryRepo repository.InventoryRepository
	productRepo   repository.ProductRepository
	lockManager   *concurrency.LockManager
	retryConfig   *concurrency.RetryConfig
	logger        *logger.Logger

	// In-memory cache for frequently accessed inventory data
	cache        map[string]*CachedInventory
	cacheMux     sync.RWMutex
	cacheTimeout time.Duration
}

// CachedInventory represents cached inventory data
type CachedInventory struct {
	ProductID   string
	Available   int
	Reserved    int
	Quantity    int
	Version     int
	LastUpdated time.Time
}

// NewEnhancedInventoryService creates a new enhanced inventory service
func NewEnhancedInventoryService(
	inventoryRepo repository.InventoryRepository,
	productRepo repository.ProductRepository,
	lockManager *concurrency.LockManager,
	logger *logger.Logger,
) EnhancedInventoryService {
	retryConfig := concurrency.DefaultRetryConfig()
	// Customize retry config for inventory operations
	retryConfig.MaxAttempts = 3
	retryConfig.InitialDelay = 100 * time.Millisecond
	retryConfig.MaxDelay = 1 * time.Second

	return &enhancedInventoryService{
		inventoryRepo: inventoryRepo,
		productRepo:   productRepo,
		lockManager:   lockManager,
		retryConfig:   retryConfig,
		logger:        logger,
		cache:         make(map[string]*CachedInventory),
		cacheTimeout:  5 * time.Minute,
	}
}

// ReserveInventoryConcurrent reserves inventory with advanced concurrency control
func (s *enhancedInventoryService) ReserveInventoryConcurrent(ctx context.Context, items []InventoryItem) error {
	if len(items) == 0 {
		return fmt.Errorf("no items to reserve")
	}

	s.logger.Info("Starting concurrent inventory reservation", "items_count", len(items))

	// Extract product IDs for locking
	productIDs := make([]string, len(items))
	for i, item := range items {
		productIDs[i] = item.ProductID
	}

	// Use distributed locking for bulk operations
	return s.lockManager.WithBulkInventoryLock(ctx, productIDs, func() error {
		return s.reserveInventoryWithRetry(ctx, items)
	})
}

// reserveInventoryWithRetry implements the actual reservation logic with retry
func (s *enhancedInventoryService) reserveInventoryWithRetry(ctx context.Context, items []InventoryItem) error {
	return concurrency.RetryWithBackoff(ctx, s.retryConfig, func() error {
		return s.reserveInventoryAtomically(ctx, items)
	}, s.logger)
}

// reserveInventoryAtomically reserves inventory in a single atomic operation
func (s *enhancedInventoryService) reserveInventoryAtomically(ctx context.Context, items []InventoryItem) error {
	// Pre-flight availability check
	for _, item := range items {
		available, err := s.CheckAvailabilityWithCache(ctx, item.ProductID, item.Quantity)
		if err != nil {
			return fmt.Errorf("availability check failed for product %s: %w", item.ProductID, err)
		}
		if !available {
			// Clear cache for this product to ensure fresh data on retry
			s.invalidateCache(item.ProductID)
			return fmt.Errorf("insufficient stock for product %s: requested %d", item.ProductID, item.Quantity)
		}
	}

	// Convert to repository format
	reservations := make([]repository.InventoryReservation, len(items))
	for i, item := range items {
		reservations[i] = repository.InventoryReservation{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	// Perform bulk reservation
	if err := s.inventoryRepo.BulkReserve(ctx, reservations); err != nil {
		// Clear cache for all affected products
		for _, item := range items {
			s.invalidateCache(item.ProductID)
		}
		return err
	}

	// Update cache with new reservation data
	s.updateCacheAfterReservation(items)

	s.logger.Info("Inventory reservation completed successfully", "items_count", len(items))
	return nil
}

// CheckAvailabilityWithCache checks inventory availability with caching
func (s *enhancedInventoryService) CheckAvailabilityWithCache(ctx context.Context, productID string, quantity int) (bool, error) {
	// Try cache first
	if cached := s.getCachedInventory(productID); cached != nil {
		s.logger.Debug("Using cached inventory data", "product_id", productID, "available", cached.Available)
		return cached.Available >= quantity, nil
	}

	// Cache miss, fetch from database
	inventory, err := s.inventoryRepo.GetByProductID(ctx, productID)
	if err != nil {
		return false, err
	}
	if inventory == nil {
		return false, nil
	}

	// Update cache
	s.setCachedInventory(productID, &CachedInventory{
		ProductID:   productID,
		Available:   inventory.Available,
		Reserved:    inventory.Reserved,
		Quantity:    inventory.Quantity,
		Version:     inventory.Version,
		LastUpdated: time.Now(),
	})

	return inventory.CanReserve(quantity), nil
}

// ProcessHighVolumeOrders handles high-volume order processing with worker pools
func (s *enhancedInventoryService) ProcessHighVolumeOrders(ctx context.Context, orders []HighVolumeOrder, workerCount int) (*HighVolumeProcessingResult, error) {
	if len(orders) == 0 {
		return &HighVolumeProcessingResult{}, nil
	}

	s.logger.Info("Starting high-volume order processing",
		"order_count", len(orders),
		"worker_count", workerCount)

	// Create worker pool
	orderChan := make(chan HighVolumeOrder, len(orders))
	resultChan := make(chan HighVolumeOrderResult, len(orders))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go s.highVolumeWorker(ctx, &wg, orderChan, resultChan)
	}

	// Send orders to workers
	go func() {
		defer close(orderChan)
		for _, order := range orders {
			orderChan <- order
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Process results
	var successful, failed []HighVolumeOrderResult
	for result := range resultChan {
		if result.Error != nil {
			failed = append(failed, result)
		} else {
			successful = append(successful, result)
		}
	}

	result := &HighVolumeProcessingResult{
		TotalOrders:       len(orders),
		SuccessfulOrders:  len(successful),
		FailedOrders:      len(failed),
		SuccessfulResults: successful,
		FailedResults:     failed,
		ProcessingTime:    time.Since(time.Now()),
	}

	s.logger.Info("High-volume order processing completed",
		"total", result.TotalOrders,
		"successful", result.SuccessfulOrders,
		"failed", result.FailedOrders)

	return result, nil
}

// highVolumeWorker processes orders in a worker pool
func (s *enhancedInventoryService) highVolumeWorker(ctx context.Context, wg *sync.WaitGroup, orderChan <-chan HighVolumeOrder, resultChan chan<- HighVolumeOrderResult) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case order, ok := <-orderChan:
			if !ok {
				return
			}

			result := HighVolumeOrderResult{
				OrderID:   order.OrderID,
				ProductID: order.ProductID,
				Quantity:  order.Quantity,
				StartTime: time.Now(),
			}

			// Process single order
			err := s.ReserveInventoryConcurrent(ctx, []InventoryItem{
				{
					ProductID: order.ProductID,
					Quantity:  order.Quantity,
				},
			})

			result.EndTime = time.Now()
			result.ProcessingTime = result.EndTime.Sub(result.StartTime)
			result.Error = err

			if err != nil {
				s.logger.Warn("High-volume order processing failed",
					"order_id", order.OrderID,
					"product_id", order.ProductID,
					"error", err)
			}

			resultChan <- result
		}
	}
}

// Cache management methods
func (s *enhancedInventoryService) getCachedInventory(productID string) *CachedInventory {
	s.cacheMux.RLock()
	defer s.cacheMux.RUnlock()

	cached, exists := s.cache[productID]
	if !exists {
		return nil
	}

	// Check if cache is expired
	if time.Since(cached.LastUpdated) > s.cacheTimeout {
		return nil
	}

	return cached
}

func (s *enhancedInventoryService) setCachedInventory(productID string, inventory *CachedInventory) {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()
	s.cache[productID] = inventory
}

func (s *enhancedInventoryService) invalidateCache(productID string) {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()
	delete(s.cache, productID)
}

func (s *enhancedInventoryService) updateCacheAfterReservation(items []InventoryItem) {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()

	for _, item := range items {
		if cached, exists := s.cache[item.ProductID]; exists {
			// Update cached values
			cached.Reserved += item.Quantity
			cached.Available = cached.Quantity - cached.Reserved
			cached.LastUpdated = time.Now()
		}
	}
}

// Base InventoryService methods implementation
func (s *enhancedInventoryService) CheckAvailability(ctx context.Context, productID string, quantity int) (bool, error) {
	return s.CheckAvailabilityWithCache(ctx, productID, quantity)
}

func (s *enhancedInventoryService) ReserveInventory(ctx context.Context, items []InventoryItem) error {
	return s.ReserveInventoryConcurrent(ctx, items)
}

func (s *enhancedInventoryService) ReleaseInventory(ctx context.Context, items []InventoryItem) error {
	if len(items) == 0 {
		return fmt.Errorf("no items to release")
	}

	// Extract product IDs for locking
	productIDs := make([]string, len(items))
	for i, item := range items {
		productIDs[i] = item.ProductID
	}

	// Use distributed locking for bulk operations
	return s.lockManager.WithBulkInventoryLock(ctx, productIDs, func() error {
		// Convert to repository format
		reservations := make([]repository.InventoryReservation, len(items))
		for i, item := range items {
			reservations[i] = repository.InventoryReservation{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			}
		}

		// Perform bulk release with retry
		return concurrency.RetryWithBackoff(ctx, s.retryConfig, func() error {
			if err := s.inventoryRepo.BulkRelease(ctx, reservations); err != nil {
				// Clear cache for all affected products
				for _, item := range items {
					s.invalidateCache(item.ProductID)
				}
				return err
			}
			return nil
		}, s.logger)
	})
}

func (s *enhancedInventoryService) UpdateStock(ctx context.Context, productID string, quantity int) error {
	return s.lockManager.WithInventoryLock(ctx, productID, func() error {
		return concurrency.RetryWithBackoff(ctx, s.retryConfig, func() error {
			if err := s.inventoryRepo.UpdateStock(ctx, productID, quantity); err != nil {
				s.invalidateCache(productID)
				return err
			}
			s.invalidateCache(productID) // Always invalidate after update
			return nil
		}, s.logger)
	})
}

func (s *enhancedInventoryService) GetLowStockAlert(ctx context.Context, threshold int) (*LowStockResponse, error) {
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
		Items:     alerts,
		Count:     len(alerts),
	}, nil
}
