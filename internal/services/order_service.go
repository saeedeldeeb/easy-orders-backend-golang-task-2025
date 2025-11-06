package services

import (
	"context"
	"errors"
	"fmt"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/logger"
)

// orderService implements OrderService interface
type orderService struct {
	orderRepo     repository.OrderRepository
	orderItemRepo repository.OrderItemRepository
	productRepo   repository.ProductRepository
	inventoryRepo repository.InventoryRepository
	userRepo      repository.UserRepository
	logger        *logger.Logger
}

// NewOrderService creates a new order service
func NewOrderService(
	orderRepo repository.OrderRepository,
	orderItemRepo repository.OrderItemRepository,
	productRepo repository.ProductRepository,
	inventoryRepo repository.InventoryRepository,
	userRepo repository.UserRepository,
	logger *logger.Logger,
) OrderService {
	return &orderService{
		orderRepo:     orderRepo,
		orderItemRepo: orderItemRepo,
		productRepo:   productRepo,
		inventoryRepo: inventoryRepo,
		userRepo:      userRepo,
		logger:        logger,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error) {
	s.logger.Info("Creating order", "user_id", req.UserID, "items_count", len(req.Items))

	// Validate request
	if req.UserID == "" {
		return nil, errors.New("user ID is required")
	}
	if len(req.Items) == 0 {
		return nil, errors.New("order must have at least one item")
	}

	// Check if a user exists
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		s.logger.Error("Failed to get user for order", "error", err, "user_id", req.UserID)
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Validate and calculate order total
	var totalAmount float64
	orderItems := make([]*models.OrderItem, 0, len(req.Items))

	for _, item := range req.Items {
		if item.ProductID == "" {
			return nil, errors.New("product ID is required for all items")
		}
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("quantity must be greater than 0 for product %s", item.ProductID)
		}

		// Get product
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			s.logger.Error("Failed to get product for order", "error", err, "product_id", item.ProductID)
			return nil, err
		}
		if product == nil {
			return nil, fmt.Errorf("product %s not found", item.ProductID)
		}
		if !product.IsActive {
			return nil, fmt.Errorf("product %s is not available", item.ProductID)
		}

		// Check inventory availability
		inventory, err := s.inventoryRepo.GetByProductID(ctx, item.ProductID)
		if err != nil {
			s.logger.Error("Failed to get inventory for order", "error", err, "product_id", item.ProductID)
			return nil, err
		}
		if inventory == nil || !inventory.CanReserve(item.Quantity) {
			return nil, fmt.Errorf("insufficient stock for product %s", item.ProductID)
		}

		// Create an order item
		unitPrice := product.Price
		totalPrice := unitPrice * float64(item.Quantity)
		totalAmount += totalPrice

		orderItem := &models.OrderItem{
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			UnitPrice:  unitPrice,
			TotalPrice: totalPrice,
		}
		orderItems = append(orderItems, orderItem)
	}

	// Create order
	order := &models.Order{
		UserID:      req.UserID,
		Status:      models.OrderStatusPending,
		TotalAmount: totalAmount,
		Currency:    "USD",
		Notes:       req.Notes,
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		s.logger.Error("Failed to create order", "error", err, "user_id", req.UserID)
		return nil, err
	}

	// Create order items
	for _, orderItem := range orderItems {
		orderItem.OrderID = order.ID
	}

	if err := s.orderItemRepo.CreateBatch(ctx, orderItems); err != nil {
		s.logger.Error("Failed to create order items", "error", err, "order_id", order.ID)
		return nil, err
	}

	s.logger.Info("Order created successfully", "order_id", order.ID, "total", totalAmount)

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

func (s *orderService) GetOrder(ctx context.Context, id string) (*OrderResponse, error) {
	s.logger.Debug("Getting order", "id", id)

	if id == "" {
		return nil, errors.New("order ID is required")
	}

	order, err := s.orderRepo.GetByIDWithItems(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get order", "error", err, "id", id)
		return nil, err
	}

	if order == nil {
		return nil, errors.New("order not found")
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
		return nil, errors.New("order ID is required")
	}

	// Get existing order
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get order for status update", "error", err, "id", id)
		return nil, err
	}

	if order == nil {
		return nil, errors.New("order not found")
	}

	// Check if status transition is valid
	if !order.CanTransitionTo(status) {
		return nil, fmt.Errorf("cannot transition from %s to %s", order.Status, status)
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
		return errors.New("order ID is required")
	}

	// Get existing order
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get order for cancellation", "error", err, "id", id)
		return err
	}

	if order == nil {
		return errors.New("order not found")
	}

	// Check if order can be cancelled
	if !order.IsCancellable() {
		return fmt.Errorf("order in status %s cannot be cancelled", order.Status)
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
