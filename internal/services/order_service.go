package services

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// orderService implements OrderService interface
type orderService struct {
	db            *database.DB
	orderRepo     repository.OrderRepository
	orderItemRepo repository.OrderItemRepository
	productRepo   repository.ProductRepository
	inventoryRepo repository.InventoryRepository
	userRepo      repository.UserRepository
	inventoryServ InventoryService
	logger        *logger.Logger
}

// NewOrderService creates a new order service
func NewOrderService(
	db *database.DB,
	orderRepo repository.OrderRepository,
	orderItemRepo repository.OrderItemRepository,
	productRepo repository.ProductRepository,
	inventoryRepo repository.InventoryRepository,
	userRepo repository.UserRepository,
	inventoryServ InventoryService,
	logger *logger.Logger,
) OrderService {
	return &orderService{
		db:            db,
		orderRepo:     orderRepo,
		orderItemRepo: orderItemRepo,
		productRepo:   productRepo,
		inventoryRepo: inventoryRepo,
		userRepo:      userRepo,
		inventoryServ: inventoryServ,
		logger:        logger,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error) {
	s.logger.Info("Creating order", "user_id", req.UserID, "items_count", len(req.Items))

	// Validate request
	if req.UserID == "" {
		return nil, errors.NewValidationError("user ID is required")
	}
	if len(req.Items) == 0 {
		return nil, errors.NewValidationError("order must have at least one item")
	}

	// Check if user exists (outside transaction for better performance)
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		s.logger.Error("Failed to get user for order", "error", err, "user_id", req.UserID)
		return nil, err
	}
	if user == nil {
		return nil, errors.NewNotFoundError("user")
	}

	var order *models.Order
	var orderItems []*models.OrderItem
	var inventoryItems []InventoryItem

	// Use database transaction for atomicity
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Create transaction context
		txCtx := context.WithValue(ctx, "db_tx", tx)

		// Validate products and calculate order total
		var totalAmount float64
		orderItems = make([]*models.OrderItem, 0, len(req.Items))
		inventoryItems = make([]InventoryItem, 0, len(req.Items))

		for _, item := range req.Items {
			if item.ProductID == "" {
				return errors.NewValidationError("product ID is required for all items")
			}
			if item.Quantity <= 0 {
				return errors.NewValidationErrorWithDetails(
					"invalid quantity",
					fmt.Sprintf("quantity must be greater than 0 for product %s", item.ProductID))
			}

			// Get product using transaction context
			var product models.Product
			if err := tx.WithContext(txCtx).First(&product, "id = ?", item.ProductID).Error; err != nil {
				if stderrors.Is(err, gorm.ErrRecordNotFound) {
					return errors.NewNotFoundErrorWithID("product", item.ProductID)
				}
				s.logger.Error("Failed to get product for order", "error", err, "product_id", item.ProductID)
				return err
			}

			if !product.IsActive {
				return errors.NewBusinessError(fmt.Sprintf("product %s is not available", item.ProductID))
			}

			// Check and lock inventory using SELECT FOR UPDATE
			// This prevents race conditions by locking the inventory row until transaction commits
			var inventory models.Inventory
			if err := tx.WithContext(txCtx).Clauses(
				// FOR UPDATE locks the row for the duration of the transaction
				clause.Locking{Strength: "UPDATE"},
			).First(&inventory, "product_id = ?", item.ProductID).Error; err != nil {
				if stderrors.Is(err, gorm.ErrRecordNotFound) {
					return errors.NewNotFoundErrorWithID("inventory", item.ProductID)
				}
				// Check if it's a lock timeout error
				if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "lock") {
					s.logger.Warn("Lock timeout while acquiring inventory", "error", err, "product_id", item.ProductID)
					return errors.NewLockTimeoutError("inventory", item.ProductID)
				}
				s.logger.Error("Failed to get and lock inventory", "error", err, "product_id", item.ProductID)
				return err
			}

			// Check if sufficient stock is available
			if !inventory.CanReserve(item.Quantity) {
				return errors.NewInsufficientStockError(item.ProductID, item.Quantity, inventory.Available)
			}

			// Calculate prices
			unitPrice := product.Price
			totalPrice := unitPrice * float64(item.Quantity)
			totalAmount += totalPrice

			// Prepare order item
			orderItem := &models.OrderItem{
				ProductID:  item.ProductID,
				Quantity:   item.Quantity,
				UnitPrice:  unitPrice,
				TotalPrice: totalPrice,
			}
			orderItems = append(orderItems, orderItem)

			// Track inventory to reserve
			inventoryItems = append(inventoryItems, InventoryItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			})
		}

		// Create order within transaction
		order = &models.Order{
			UserID:      req.UserID,
			Status:      models.OrderStatusPending,
			TotalAmount: totalAmount,
			Currency:    "USD",
			Notes:       req.Notes,
		}

		if err := tx.WithContext(txCtx).Create(order).Error; err != nil {
			s.logger.Error("Failed to create order", "error", err, "user_id", req.UserID)
			return err
		}

		// Set order ID for all items and create them
		for _, orderItem := range orderItems {
			orderItem.OrderID = order.ID
		}

		if err := tx.WithContext(txCtx).Create(&orderItems).Error; err != nil {
			s.logger.Error("Failed to create order items", "error", err, "order_id", order.ID)
			return err
		}

		// Reserve inventory within the same transaction
		// Use bulk reserve for better performance
		reservations := make([]repository.InventoryReservation, len(inventoryItems))
		for i, item := range inventoryItems {
			reservations[i] = repository.InventoryReservation{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			}
		}

		// Reserve using the transaction context
		if err := s.reserveStockInTransaction(tx, txCtx, reservations); err != nil {
			s.logger.Error("Failed to reserve inventory", "error", err, "order_id", order.ID)
			return fmt.Errorf("failed to reserve inventory: %w", err)
		}

		s.logger.Info("Order created and inventory reserved successfully",
			"order_id", order.ID, "total", totalAmount, "items_count", len(orderItems))

		return nil
	})

	if err != nil {
		s.logger.Error("Transaction failed during order creation", "error", err, "user_id", req.UserID)
		return nil, err
	}

	// Convert to response format
	responseItems := make([]OrderItem, len(orderItems))
	for i, item := range orderItems {
		responseItems[i] = OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
	}

	return &OrderResponse{
		ID:     order.ID,
		UserID: order.UserID,
		Status: order.Status,
		Items:  responseItems,
		Total:  order.TotalAmount,
	}, nil
}

// reserveStockInTransaction reserves inventory within an existing transaction
func (s *orderService) reserveStockInTransaction(tx *gorm.DB, ctx context.Context, items []repository.InventoryReservation) error {
	for _, item := range items {
		var inventory models.Inventory
		if err := tx.WithContext(ctx).First(&inventory, "product_id = ?", item.ProductID).Error; err != nil {
			return errors.NewDatabaseError("failed to get inventory for reservation", err)
		}

		// Reserve the stock
		oldVersion := inventory.Version
		if err := inventory.Reserve(item.Quantity); err != nil {
			return errors.NewInsufficientStockError(item.ProductID, item.Quantity, inventory.Available)
		}
		inventory.Version++

		// Update with optimistic locking
		result := tx.WithContext(ctx).Model(&inventory).
			Where("product_id = ? AND version = ?", item.ProductID, oldVersion).
			Updates(map[string]interface{}{
				"reserved":  inventory.Reserved,
				"available": inventory.Available,
				"version":   inventory.Version,
			})

		if result.Error != nil {
			return errors.NewDatabaseError("failed to update inventory reservation", result.Error)
		}

		// Optimistic lock failure - version mismatch
		if result.RowsAffected == 0 {
			s.logger.Warn("Optimistic lock failed during inventory reservation",
				"product_id", item.ProductID, "expected_version", oldVersion)
			return errors.NewStockReservationConflictError(item.ProductID,
				stderrors.New("inventory was modified by another transaction"))
		}

		s.logger.Debug("Stock reserved in transaction", "product_id", item.ProductID, "quantity", item.Quantity)
	}

	return nil
}

func (s *orderService) GetOrder(ctx context.Context, id string) (*OrderResponse, error) {
	s.logger.Debug("Getting order", "id", id)

	if id == "" {
		return nil, errors.NewValidationError("order ID is required")
	}

	order, err := s.orderRepo.GetByIDWithItems(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get order", "error", err, "id", id)
		return nil, err
	}

	if order == nil {
		return nil, errors.NewNotFoundErrorWithID("order", id)
	}

	// Convert order items to response format
	responseItems := make([]OrderItem, len(order.Items))
	for i, item := range order.Items {
		responseItems[i] = OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
	}

	return &OrderResponse{
		ID:     order.ID,
		UserID: order.UserID,
		Status: order.Status,
		Items:  responseItems,
		Total:  order.TotalAmount,
	}, nil
}

func (s *orderService) UpdateOrderStatus(ctx context.Context, id string, status models.OrderStatus) (*OrderResponse, error) {
	s.logger.Info("Updating order status", "id", id, "status", status)

	if id == "" {
		return nil, errors.NewValidationError("order ID is required")
	}

	// Get existing order
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get order for status update", "error", err, "id", id)
		return nil, err
	}

	if order == nil {
		return nil, errors.NewNotFoundErrorWithID("order", id)
	}

	// Check if status transition is valid
	if !order.CanTransitionTo(status) {
		return nil, errors.NewInvalidTransitionError(string(order.Status), string(status))
	}

	// Update order status
	if err := s.orderRepo.UpdateStatus(ctx, id, status); err != nil {
		s.logger.Error("Failed to update order status", "error", err, "id", id, "status", status)
		return nil, err
	}

	s.logger.Info("Order status updated successfully", "id", id, "new_status", status)

	// Get updated order with items
	updatedOrder, err := s.orderRepo.GetByIDWithItems(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get updated order", "error", err, "id", id)
		return nil, err
	}

	// Convert to response format
	responseItems := make([]OrderItem, len(updatedOrder.Items))
	for i, item := range updatedOrder.Items {
		responseItems[i] = OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
		}
	}

	return &OrderResponse{
		ID:     updatedOrder.ID,
		UserID: updatedOrder.UserID,
		Status: updatedOrder.Status,
		Items:  responseItems,
		Total:  updatedOrder.TotalAmount,
	}, nil
}

func (s *orderService) CancelOrder(ctx context.Context, id string) error {
	s.logger.Info("Cancelling order", "id", id)

	if id == "" {
		return errors.NewValidationError("order ID is required")
	}

	// Get existing order
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get order for cancellation", "error", err, "id", id)
		return err
	}

	if order == nil {
		return errors.NewNotFoundErrorWithID("order", id)
	}

	// Check if order can be cancelled
	if !order.IsCancellable() {
		return errors.NewBusinessError(fmt.Sprintf("order in status %s cannot be cancelled", order.Status))
	}

	// Update order status to cancelled
	if err := s.orderRepo.UpdateStatus(ctx, id, models.OrderStatusCancelled); err != nil {
		s.logger.Error("Failed to cancel order", "error", err, "id", id)
		return err
	}

	s.logger.Info("Order cancelled successfully", "id", id)
	return nil
}

func (s *orderService) ListOrders(ctx context.Context, req ListOrdersRequest) (*ListOrdersResponse, error) {
	s.logger.Debug("Listing orders", "page", req.Page, "limit", req.Limit, "status", req.Status)

	// Set default limit if not provided
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20 // Default limit
	}

	// Set default page to 1 if not provided or invalid
	page := req.Page
	if page < 1 {
		page = 1
	}

	// Calculate offset from page number
	offset := (page - 1) * limit

	// Get paginated orders
	var orders []*models.Order
	var err error

	if req.Status != "" {
		orders, err = s.orderRepo.ListByStatus(ctx, req.Status, offset, limit)
	} else {
		orders, err = s.orderRepo.List(ctx, offset, limit)
	}

	if err != nil {
		s.logger.Error("Failed to list orders", "error", err)
		return nil, err
	}

	// Get total count
	var totalCount int64
	if req.Status != "" {
		totalCount, err = s.orderRepo.CountByStatus(ctx, req.Status)
	} else {
		totalCount, err = s.orderRepo.Count(ctx)
	}

	if err != nil {
		s.logger.Error("Failed to count orders", "error", err)
		return nil, err
	}

	// Convert to response format
	orderResponses := make([]*OrderResponse, len(orders))
	for i, order := range orders {
		responseItems := make([]OrderItem, len(order.Items))
		for j, item := range order.Items {
			responseItems[j] = OrderItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
			}
		}

		orderResponses[i] = &OrderResponse{
			ID:     order.ID,
			UserID: order.UserID,
			Status: order.Status,
			Items:  responseItems,
			Total:  order.TotalAmount,
		}
	}

	s.logger.Debug("Orders listed successfully", "count", len(orderResponses))

	return &ListOrdersResponse{
		Orders: orderResponses,
		Page:   page,
		Limit:  limit,
		Total:  int(totalCount),
	}, nil
}
