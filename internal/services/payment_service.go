package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/logger"
)

// paymentService implements PaymentService interface
type paymentService struct {
	paymentRepo repository.PaymentRepository
	orderRepo   repository.OrderRepository
	logger      *logger.Logger
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	paymentRepo repository.PaymentRepository,
	orderRepo repository.OrderRepository,
	logger *logger.Logger,
) PaymentService {
	return &paymentService{
		paymentRepo: paymentRepo,
		orderRepo:   orderRepo,
		logger:      logger,
	}
}

func (s *paymentService) ProcessPayment(ctx context.Context, req ProcessPaymentRequest) (*PaymentResponse, error) {
	s.logger.Info("Processing payment", "order_id", req.OrderID, "amount", req.Amount, "method", req.PaymentType)

	// Validate request
	if req.OrderID == "" {
		return nil, errors.New("order ID is required")
	}
	if req.Amount <= 0 {
		return nil, errors.New("payment amount must be greater than 0")
	}
	if req.PaymentType == "" {
		return nil, errors.New("payment type is required")
	}

	// Get order
	order, err := s.orderRepo.GetByID(ctx, req.OrderID)
	if err != nil {
		s.logger.Error("Failed to get order for payment", "error", err, "order_id", req.OrderID)
		return nil, err
	}
	if order == nil {
		return nil, errors.New("order not found")
	}

	// Validate payment amount against order total
	if req.Amount != order.TotalAmount {
		return nil, fmt.Errorf("payment amount %.2f does not match order total %.2f", req.Amount, order.TotalAmount)
	}

	// Check if order is in a payable state
	if order.Status != models.OrderStatusPending && order.Status != models.OrderStatusConfirmed {
		return nil, fmt.Errorf("order in status %s cannot be paid", order.Status)
	}

	// Check for existing successful payments
	existingPayments, err := s.paymentRepo.GetByOrderID(ctx, req.OrderID)
	if err != nil {
		s.logger.Error("Failed to check existing payments", "error", err, "order_id", req.OrderID)
		return nil, err
	}

	for _, payment := range existingPayments {
		if payment.IsCompleted() {
			return nil, errors.New("order has already been paid")
		}
	}

	// Create a payment record
	payment := &models.Payment{
		OrderID:           req.OrderID,
		Amount:            req.Amount,
		Currency:          "USD",
		Status:            models.PaymentStatusPending,
		Method:            models.PaymentMethod(req.PaymentType),
		ExternalReference: req.ExternalReference,
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		s.logger.Error("Failed to create payment", "error", err, "order_id", req.OrderID)
		return nil, err
	}

	// Simulate payment processing
	success := s.simulatePaymentProcessing(ctx, payment)

	if success {
		// Mark payment as processed and completed
		payment.MarkProcessed()
		payment.MarkCompleted()

		if err := s.paymentRepo.Update(ctx, payment); err != nil {
			s.logger.Error("Failed to update payment status", "error", err, "payment_id", payment.ID)
			return nil, err
		}

		// Update order status to paid
		if err := s.orderRepo.UpdateStatus(ctx, req.OrderID, models.OrderStatusPaid); err != nil {
			s.logger.Error("Failed to update order status after payment", "error", err, "order_id", req.OrderID)
			// Don't fail the payment, just log the error
		}

		s.logger.Info("Payment processed successfully", "payment_id", payment.ID, "order_id", req.OrderID)
	} else {
		// Mark payment as failed
		payment.MarkFailed("Payment processing failed")

		if err := s.paymentRepo.Update(ctx, payment); err != nil {
			s.logger.Error("Failed to update failed payment status", "error", err, "payment_id", payment.ID)
		}

		s.logger.Warn("Payment processing failed", "payment_id", payment.ID, "order_id", req.OrderID)
		return nil, errors.New("payment processing failed")
	}

	return &PaymentResponse{
		ID:      payment.ID,
		OrderID: payment.OrderID,
		Amount:  payment.Amount,
		Status:  payment.Status,
	}, nil
}

func (s *paymentService) GetPayment(ctx context.Context, id string) (*PaymentResponse, error) {
	s.logger.Debug("Getting payment", "id", id)

	if id == "" {
		return nil, errors.New("payment ID is required")
	}

	payment, err := s.paymentRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get payment", "error", err, "id", id)
		return nil, err
	}

	if payment == nil {
		return nil, errors.New("payment not found")
	}

	return &PaymentResponse{
		ID:      payment.ID,
		OrderID: payment.OrderID,
		Amount:  payment.Amount,
		Status:  payment.Status,
	}, nil
}

func (s *paymentService) GetOrderPayments(ctx context.Context, orderID string) ([]*PaymentResponse, error) {
	s.logger.Debug("Getting order payments", "order_id", orderID)

	if orderID == "" {
		return nil, errors.New("order ID is required")
	}

	payments, err := s.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		s.logger.Error("Failed to get order payments", "error", err, "order_id", orderID)
		return nil, err
	}

	// Convert to response format
	paymentResponses := make([]*PaymentResponse, len(payments))
	for i, payment := range payments {
		paymentResponses[i] = &PaymentResponse{
			ID:      payment.ID,
			OrderID: payment.OrderID,
			Amount:  payment.Amount,
			Status:  payment.Status,
		}
	}

	s.logger.Debug("Order payments retrieved", "order_id", orderID, "count", len(paymentResponses))

	return paymentResponses, nil
}

// simulatePaymentProcessing simulates external payment processing
// In a real implementation, this would integrate with a payment gateway
func (s *paymentService) simulatePaymentProcessing(ctx context.Context, payment *models.Payment) bool {
	s.logger.Debug("Simulating payment processing", "payment_id", payment.ID, "method", payment.Method)

	// Simulate processing delay
	time.Sleep(100 * time.Millisecond)

	// Simulate 95% success rate
	// In reality, this would be determined by the payment gateway response
	success := time.Now().UnixNano()%100 < 95

	s.logger.Debug("Payment processing simulation completed", "payment_id", payment.ID, "success", success)

	return success
}
