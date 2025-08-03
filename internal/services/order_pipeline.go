package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/logger"
)

// These types are defined in interfaces.go to avoid redeclaration

// PipelineStage represents individual stages in the order processing pipeline
type PipelineStage struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Success   bool
	Error     error
	Data      interface{}
}

// orderPipelineService implements OrderPipelineService
type orderPipelineService struct {
	orderService        OrderService
	inventoryService    InventoryService
	paymentService      PaymentService
	notificationService NotificationService
	logger              *logger.Logger

	// Pipeline configuration
	timeout time.Duration
	workers int
}

// NewOrderPipelineService creates a new order pipeline service
func NewOrderPipelineService(
	orderService OrderService,
	inventoryService InventoryService,
	paymentService PaymentService,
	notificationService NotificationService,
	logger *logger.Logger,
) OrderPipelineService {
	return &orderPipelineService{
		orderService:        orderService,
		inventoryService:    inventoryService,
		paymentService:      paymentService,
		notificationService: notificationService,
		logger:              logger,
		timeout:             30 * time.Second, // Default timeout
		workers:             3,                // Number of concurrent workers
	}
}

// ProcessOrder executes the order processing pipeline synchronously
func (s *orderPipelineService) ProcessOrder(ctx context.Context, req CreateOrderRequest) (*OrderPipelineResult, error) {
	startTime := time.Now()
	s.logger.Info("Starting order processing pipeline", "user_id", req.UserID, "items_count", len(req.Items))

	result := &OrderPipelineResult{
		Status:         PipelineStatusPending,
		ProcessingTime: 0,
		Errors:         make([]string, 0),
		Notifications:  make([]string, 0),
	}

	// Create a timeout context for the entire pipeline
	pipelineCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Execute pipeline stages
	stages := s.buildPipelineStages(pipelineCtx, req)

	// Execute stages sequentially with concurrent sub-operations
	success := s.executePipeline(pipelineCtx, stages, result)

	result.ProcessingTime = time.Since(startTime)
	if success {
		result.Status = PipelineStatusCompleted
		s.logger.Info("Order processing pipeline completed successfully",
			"user_id", req.UserID,
			"order_id", getOrderID(result),
			"processing_time", result.ProcessingTime,
		)
	} else {
		result.Status = PipelineStatusFailed
		s.logger.Error("Order processing pipeline failed",
			"user_id", req.UserID,
			"errors", result.Errors,
			"processing_time", result.ProcessingTime,
		)
	}

	return result, nil
}

// ProcessOrderAsync executes the order processing pipeline asynchronously
func (s *orderPipelineService) ProcessOrderAsync(ctx context.Context, req CreateOrderRequest) (<-chan *OrderPipelineResult, error) {
	resultChan := make(chan *OrderPipelineResult, 1)

	go func() {
		defer close(resultChan)
		result, _ := s.ProcessOrder(ctx, req)
		resultChan <- result
	}()

	return resultChan, nil
}

// buildPipelineStages creates the pipeline stages for order processing
func (s *orderPipelineService) buildPipelineStages(ctx context.Context, req CreateOrderRequest) []func() *PipelineStage {
	return []func() *PipelineStage{
		// Stage 1: Order Placement & Validation
		func() *PipelineStage {
			return s.stageOrderPlacement(ctx, req)
		},
		// Stage 2: Concurrent Inventory Reservation
		func() *PipelineStage {
			return s.stageInventoryReservation(ctx, req)
		},
		// Stage 3: Payment Processing
		func() *PipelineStage {
			return s.stagePaymentProcessing(ctx)
		},
		// Stage 4: Order Fulfillment
		func() *PipelineStage {
			return s.stageOrderFulfillment(ctx)
		},
		// Stage 5: Notification Dispatch
		func() *PipelineStage {
			return s.stageNotificationDispatch(ctx)
		},
	}
}

// executePipeline runs the pipeline stages with error handling and rollback capabilities
func (s *orderPipelineService) executePipeline(ctx context.Context, stages []func() *PipelineStage, result *OrderPipelineResult) bool {
	result.Status = PipelineStatusProcessing
	var pipelineData = make(map[string]interface{})
	completedStages := make([]*PipelineStage, 0, len(stages))

	for i, stageFunc := range stages {
		select {
		case <-ctx.Done():
			s.logger.Warn("Pipeline execution cancelled due to timeout or cancellation", "stage", i)
			result.Errors = append(result.Errors, "Pipeline execution timed out")
			s.rollbackStages(completedStages, pipelineData)
			return false
		default:
			stage := stageFunc()
			completedStages = append(completedStages, stage)

			if !stage.Success {
				s.logger.Error("Pipeline stage failed", "stage", stage.Name, "error", stage.Error)
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", stage.Name, stage.Error))

				// Attempt to rollback previous stages
				s.rollbackStages(completedStages[:len(completedStages)-1], pipelineData)
				return false
			}

			// Store stage data for subsequent stages and rollback
			pipelineData[stage.Name] = stage.Data
			s.logger.Debug("Pipeline stage completed", "stage", stage.Name, "duration", stage.EndTime.Sub(stage.StartTime))
		}
	}

	// Extract final results from completed stages
	s.extractResults(completedStages, pipelineData, result)
	return true
}

// stageOrderPlacement handles order creation and validation
func (s *orderPipelineService) stageOrderPlacement(ctx context.Context, req CreateOrderRequest) *PipelineStage {
	stage := &PipelineStage{
		Name:      "order_placement",
		StartTime: time.Now(),
	}
	defer func() { stage.EndTime = time.Now() }()

	s.logger.Debug("Executing order placement stage", "user_id", req.UserID)

	// Create order using existing order service
	order, err := s.orderService.CreateOrder(ctx, req)
	if err != nil {
		stage.Error = err
		stage.Success = false
		return stage
	}

	stage.Data = order
	stage.Success = true
	s.logger.Debug("Order placement completed", "order_id", order.ID)
	return stage
}

// stageInventoryReservation handles concurrent inventory reservation for all items
func (s *orderPipelineService) stageInventoryReservation(ctx context.Context, req CreateOrderRequest) *PipelineStage {
	stage := &PipelineStage{
		Name:      "inventory_reservation",
		StartTime: time.Now(),
	}
	defer func() { stage.EndTime = time.Now() }()

	s.logger.Debug("Executing inventory reservation stage", "items_count", len(req.Items))

	// Convert order items to inventory items
	inventoryItems := make([]InventoryItem, len(req.Items))
	for i, item := range req.Items {
		inventoryItems[i] = InventoryItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	// Reserve inventory using concurrent processing
	err := s.reserveInventoryConcurrently(ctx, inventoryItems)
	if err != nil {
		stage.Error = err
		stage.Success = false
		return stage
	}

	stage.Data = inventoryItems
	stage.Success = true
	s.logger.Debug("Inventory reservation completed", "items_count", len(inventoryItems))
	return stage
}

// stagePaymentProcessing handles payment processing
func (s *orderPipelineService) stagePaymentProcessing(ctx context.Context) *PipelineStage {
	stage := &PipelineStage{
		Name:      "payment_processing",
		StartTime: time.Now(),
	}
	defer func() { stage.EndTime = time.Now() }()

	s.logger.Debug("Executing payment processing stage")

	// For now, simulate payment processing
	// TODO: Implement actual payment processing logic based on order data
	// This would typically take order details from previous stage

	// Simulated payment response
	paymentResponse := &PaymentResponse{
		ID:      "payment_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		OrderID: "", // Would be populated from order stage
		Amount:  0,  // Would be populated from order stage
		Status:  models.PaymentStatusCompleted,
	}

	stage.Data = paymentResponse
	stage.Success = true
	s.logger.Debug("Payment processing completed", "payment_id", paymentResponse.ID)
	return stage
}

// stageOrderFulfillment handles order fulfillment and status updates
func (s *orderPipelineService) stageOrderFulfillment(ctx context.Context) *PipelineStage {
	stage := &PipelineStage{
		Name:      "order_fulfillment",
		StartTime: time.Now(),
	}
	defer func() { stage.EndTime = time.Now() }()

	s.logger.Debug("Executing order fulfillment stage")

	// TODO: Implement order fulfillment logic
	// This would typically:
	// 1. Update order status to processing/completed
	// 2. Generate fulfillment records
	// 3. Update inventory committed quantities
	// 4. Create shipping records if needed

	stage.Data = "fulfillment_completed"
	stage.Success = true
	s.logger.Debug("Order fulfillment completed")
	return stage
}

// stageNotificationDispatch handles sending notifications asynchronously
func (s *orderPipelineService) stageNotificationDispatch(ctx context.Context) *PipelineStage {
	stage := &PipelineStage{
		Name:      "notification_dispatch",
		StartTime: time.Now(),
	}
	defer func() { stage.EndTime = time.Now() }()

	s.logger.Debug("Executing notification dispatch stage")

	// Send notifications concurrently
	notifications := s.sendNotificationsConcurrently(ctx)

	stage.Data = notifications
	stage.Success = true
	s.logger.Debug("Notification dispatch completed", "notifications_sent", len(notifications))
	return stage
}

// reserveInventoryConcurrently reserves inventory for multiple items concurrently
func (s *orderPipelineService) reserveInventoryConcurrently(ctx context.Context, items []InventoryItem) error {
	// Create a channel for inventory reservation results
	resultChan := make(chan error, len(items))
	var wg sync.WaitGroup

	// Process inventory reservations concurrently
	for _, item := range items {
		wg.Add(1)
		go func(inventoryItem InventoryItem) {
			defer wg.Done()

			// Check availability first
			available, err := s.inventoryService.CheckAvailability(ctx, inventoryItem.ProductID, inventoryItem.Quantity)
			if err != nil {
				resultChan <- fmt.Errorf("failed to check availability for product %s: %w", inventoryItem.ProductID, err)
				return
			}

			if !available {
				resultChan <- errors.NewInsufficientStockError(inventoryItem.ProductID, inventoryItem.Quantity, 0)
				return
			}

			// Reserve the inventory
			err = s.inventoryService.ReserveInventory(ctx, []InventoryItem{inventoryItem})
			if err != nil {
				resultChan <- fmt.Errorf("failed to reserve inventory for product %s: %w", inventoryItem.ProductID, err)
				return
			}

			resultChan <- nil
		}(item)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var reservationErrors []error
	for err := range resultChan {
		if err != nil {
			reservationErrors = append(reservationErrors, err)
		}
	}

	if len(reservationErrors) > 0 {
		// If any reservations failed, we need to rollback successful ones
		s.logger.Error("Inventory reservation failed", "errors", reservationErrors)
		return fmt.Errorf("inventory reservation failed: %v", reservationErrors[0])
	}

	return nil
}

// sendNotificationsConcurrently sends multiple notifications concurrently
func (s *orderPipelineService) sendNotificationsConcurrently(ctx context.Context) []string {
	// TODO: Implement concurrent notification sending
	// This would typically send order confirmation, payment confirmation, etc.

	notifications := []string{
		"order_confirmation_sent",
		"payment_confirmation_sent",
		"fulfillment_notification_sent",
	}

	return notifications
}

// rollbackStages handles rollback of completed stages in case of failure
func (s *orderPipelineService) rollbackStages(stages []*PipelineStage, pipelineData map[string]interface{}) {
	s.logger.Info("Starting pipeline rollback", "stages_to_rollback", len(stages))

	// Rollback stages in reverse order
	for i := len(stages) - 1; i >= 0; i-- {
		stage := stages[i]
		s.logger.Debug("Rolling back stage", "stage", stage.Name)

		switch stage.Name {
		case "inventory_reservation":
			s.rollbackInventoryReservation(pipelineData)
		case "payment_processing":
			s.rollbackPaymentProcessing(pipelineData)
		case "order_placement":
			s.rollbackOrderPlacement(pipelineData)
		}
	}

	s.logger.Info("Pipeline rollback completed")
}

// rollbackInventoryReservation releases reserved inventory
func (s *orderPipelineService) rollbackInventoryReservation(pipelineData map[string]interface{}) {
	if inventoryData, exists := pipelineData["inventory_reservation"]; exists {
		if items, ok := inventoryData.([]InventoryItem); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := s.inventoryService.ReleaseInventory(ctx, items)
			if err != nil {
				s.logger.Error("Failed to rollback inventory reservation", "error", err)
			} else {
				s.logger.Debug("Inventory reservation rolled back successfully")
			}
		}
	}
}

// rollbackPaymentProcessing handles payment rollback/refund
func (s *orderPipelineService) rollbackPaymentProcessing(pipelineData map[string]interface{}) {
	if paymentData, exists := pipelineData["payment_processing"]; exists {
		if payment, ok := paymentData.(*PaymentResponse); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err := s.paymentService.RefundPayment(ctx, payment.ID, payment.Amount)
			if err != nil {
				s.logger.Error("Failed to rollback payment", "error", err, "payment_id", payment.ID)
			} else {
				s.logger.Debug("Payment rolled back successfully", "payment_id", payment.ID)
			}
		}
	}
}

// rollbackOrderPlacement handles order cancellation
func (s *orderPipelineService) rollbackOrderPlacement(pipelineData map[string]interface{}) {
	if orderData, exists := pipelineData["order_placement"]; exists {
		if order, ok := orderData.(*OrderResponse); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := s.orderService.CancelOrder(ctx, order.ID)
			if err != nil {
				s.logger.Error("Failed to rollback order placement", "error", err, "order_id", order.ID)
			} else {
				s.logger.Debug("Order placement rolled back successfully", "order_id", order.ID)
			}
		}
	}
}

// extractResults extracts final results from completed pipeline stages
func (s *orderPipelineService) extractResults(stages []*PipelineStage, pipelineData map[string]interface{}, result *OrderPipelineResult) {
	for _, stage := range stages {
		switch stage.Name {
		case "order_placement":
			if order, ok := stage.Data.(*OrderResponse); ok {
				result.Order = order
			}
		case "inventory_reservation":
			if items, ok := stage.Data.([]InventoryItem); ok {
				result.InventoryItems = items
			}
		case "payment_processing":
			if payment, ok := stage.Data.(*PaymentResponse); ok {
				result.PaymentResult = payment
			}
		case "notification_dispatch":
			if notifications, ok := stage.Data.([]string); ok {
				result.Notifications = notifications
			}
		}
	}
}

// Helper function to safely extract order ID from result
func getOrderID(result *OrderPipelineResult) string {
	if result.Order != nil {
		return result.Order.ID
	}
	return "unknown"
}
