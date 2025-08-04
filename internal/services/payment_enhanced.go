package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/concurrency"
	"easy-orders-backend/pkg/logger"
	"easy-orders-backend/pkg/payments"
	"easy-orders-backend/pkg/workers"
)

// EnhancedPaymentService provides advanced payment processing with idempotency and retry logic
type EnhancedPaymentService struct {
	paymentRepo        repository.PaymentRepository
	orderRepo          repository.OrderRepository
	gatewayManager     *payments.PaymentGatewayManager
	idempotencyManager *payments.IdempotencyManager
	circuitBreaker     *payments.CircuitBreakerManager
	poolManager        *workers.PoolManager
	lockManager        concurrency.LockManager
	logger             *logger.Logger
}

// NewEnhancedPaymentService creates a new enhanced payment service
func NewEnhancedPaymentService(
	paymentRepo repository.PaymentRepository,
	orderRepo repository.OrderRepository,
	gatewayManager *payments.PaymentGatewayManager,
	idempotencyManager *payments.IdempotencyManager,
	circuitBreaker *payments.CircuitBreakerManager,
	poolManager *workers.PoolManager,
	lockManager concurrency.LockManager,
	logger *logger.Logger,
) *EnhancedPaymentService {
	service := &EnhancedPaymentService{
		paymentRepo:        paymentRepo,
		orderRepo:          orderRepo,
		gatewayManager:     gatewayManager,
		idempotencyManager: idempotencyManager,
		circuitBreaker:     circuitBreaker,
		poolManager:        poolManager,
		lockManager:        lockManager,
		logger:             logger,
	}

	// Job executor registration is handled differently in this implementation

	return service
}

// ProcessPaymentIdempotent processes a payment with full idempotency and retry support
func (eps *EnhancedPaymentService) ProcessPaymentIdempotent(ctx context.Context, req *payments.PaymentRequest) (*payments.PaymentResult, error) {
	eps.logger.Info("Processing idempotent payment",
		"order_id", req.OrderID,
		"idempotency_key", req.IdempotencyKey,
		"amount", req.Amount,
		"gateway", req.Gateway)

	// Check idempotency
	if existingRecord, found := eps.idempotencyManager.CheckIdempotency(ctx, req); found {
		eps.logger.Info("Found existing payment request",
			"idempotency_key", req.IdempotencyKey,
			"payment_id", existingRecord.PaymentID,
			"status", existingRecord.Status)

		if existingRecord.Result != nil {
			return existingRecord.Result, nil
		}

		// If no result yet, check payment status in database
		return eps.getPaymentResult(ctx, existingRecord.PaymentID)
	}

	// Validate request
	if err := eps.validatePaymentRequest(ctx, req); err != nil {
		eps.logger.Error("Payment request validation failed", "error", err, "order_id", req.OrderID)
		return nil, err
	}

	// Create payment record with idempotency protection
	payment, err := eps.createPaymentRecord(ctx, req)
	if err != nil {
		eps.logger.Error("Failed to create payment record", "error", err, "order_id", req.OrderID)
		return nil, err
	}

	// Store idempotency record
	eps.idempotencyManager.StoreIdempotencyRecord(ctx, req, payment.ID, "processing")

	// Process payment with retries
	result := eps.processPaymentWithRetries(ctx, req, payment)

	// Update idempotency record with final result
	status := "failed"
	if result.Success {
		status = "completed"
	}
	eps.idempotencyManager.UpdateIdempotencyRecord(ctx, req.IdempotencyKey, status, result)

	return result, nil
}

// processPaymentWithRetries handles the payment processing with retry logic
func (eps *EnhancedPaymentService) processPaymentWithRetries(ctx context.Context, req *payments.PaymentRequest, payment *models.Payment) *payments.PaymentResult {
	startTime := time.Now()
	retryPolicy := req.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = payments.DefaultRetryPolicy()
	}

	result := &payments.PaymentResult{
		PaymentID:      payment.ID,
		IdempotencyKey: req.IdempotencyKey,
		Amount:         req.Amount,
		Currency:       req.Currency,
		CreatedAt:      startTime,
		Attempts:       make([]payments.PaymentAttempt, 0),
	}

	for attempt := 1; attempt <= retryPolicy.MaxAttempts; attempt++ {
		eps.logger.Debug("Processing payment attempt",
			"payment_id", payment.ID,
			"attempt", attempt,
			"max_attempts", retryPolicy.MaxAttempts)

		// Update payment attempt count
		payment.IncrementAttempt()
		if err := eps.paymentRepo.Update(ctx, payment); err != nil {
			eps.logger.Error("Failed to update payment attempt count", "error", err, "payment_id", payment.ID)
		}

		// Process single attempt
		attemptResult := eps.processSingleAttempt(ctx, req, payment, attempt)
		result.Attempts = append(result.Attempts, attemptResult)
		result.AttemptCount = attempt

		if attemptResult.Success {
			// Payment succeeded
			result.Success = true
			result.Status = "completed"
			now := time.Now()
			result.CompletedAt = &now
			result.TotalProcessingTime = time.Since(startTime)

			// Mark payment as completed
			payment.MarkCompleted()
			payment.GatewayTxnID = attemptResult.GatewayResponse["transaction_id"].(string)
			if err := eps.paymentRepo.Update(ctx, payment); err != nil {
				eps.logger.Error("Failed to update completed payment", "error", err, "payment_id", payment.ID)
			}

			// Update order status
			if err := eps.orderRepo.UpdateStatus(ctx, req.OrderID, models.OrderStatusPaid); err != nil {
				eps.logger.Error("Failed to update order status after payment", "error", err, "order_id", req.OrderID)
			}

			eps.logger.Info("Payment completed successfully",
				"payment_id", payment.ID,
				"attempt", attempt,
				"processing_time_ms", result.TotalProcessingTime.Milliseconds())

			return result
		}

		// Payment failed - check if we should retry
		if attempt >= retryPolicy.MaxAttempts {
			eps.logger.Warn("Payment exhausted all retry attempts",
				"payment_id", payment.ID,
				"attempts", attempt)
			break
		}

		// Check if failure type is retriable
		if !retryPolicy.IsRetriable(attemptResult.FailureType) {
			eps.logger.Warn("Payment failed with non-retriable error",
				"payment_id", payment.ID,
				"failure_type", attemptResult.FailureType,
				"message", attemptResult.FailureMessage)
			break
		}

		// Calculate retry delay
		retryDelay := retryPolicy.CalculateNextRetryDelay(attempt)
		if retryDelay > 0 {
			nextRetryAt := time.Now().Add(retryDelay)
			payment.SetNextRetryAt(nextRetryAt)

			eps.logger.Info("Scheduling payment retry",
				"payment_id", payment.ID,
				"retry_delay_ms", retryDelay.Milliseconds(),
				"next_retry_at", nextRetryAt)

			// Update payment with next retry time
			if err := eps.paymentRepo.Update(ctx, payment); err != nil {
				eps.logger.Error("Failed to update payment retry schedule", "error", err, "payment_id", payment.ID)
			}

			// For immediate retries within the same process, wait
			if retryDelay <= 30*time.Second {
				time.Sleep(retryDelay)
			} else {
				// For longer delays, schedule async retry
				eps.scheduleAsyncRetry(ctx, req, payment, nextRetryAt)

				// Return partial result for now
				result.Status = "retrying"
				result.TotalProcessingTime = time.Since(startTime)
				return result
			}
		}
	}

	// All retries exhausted or non-retriable failure
	lastAttempt := result.Attempts[len(result.Attempts)-1]
	result.Success = false
	result.Status = "failed"
	result.FinalFailureType = lastAttempt.FailureType
	result.FinalFailureMessage = lastAttempt.FailureMessage
	result.TotalProcessingTime = time.Since(startTime)
	now := time.Now()
	result.CompletedAt = &now

	// Mark payment as failed
	payment.MarkFailed(lastAttempt.FailureMessage)
	if err := eps.paymentRepo.Update(ctx, payment); err != nil {
		eps.logger.Error("Failed to update failed payment", "error", err, "payment_id", payment.ID)
	}

	eps.logger.Error("Payment failed after all retries",
		"payment_id", payment.ID,
		"final_failure_type", result.FinalFailureType,
		"final_failure_message", result.FinalFailureMessage)

	return result
}

// processSingleAttempt processes a single payment attempt
func (eps *EnhancedPaymentService) processSingleAttempt(ctx context.Context, req *payments.PaymentRequest, payment *models.Payment, attemptNumber int) payments.PaymentAttempt {
	startTime := time.Now()

	attempt := payments.PaymentAttempt{
		AttemptNumber: attemptNumber,
		Gateway:       req.Gateway,
		StartedAt:     startTime,
	}

	// Get circuit breaker for the gateway
	circuitBreaker := eps.circuitBreaker.GetCircuitBreaker(req.Gateway)

	// Prepare gateway request
	gatewayReq := &payments.GatewayPaymentRequest{
		Amount:          req.Amount,
		Currency:        req.Currency,
		PaymentMethod:   req.PaymentMethod,
		IdempotencyKey:  req.IdempotencyKey,
		OrderReference:  req.OrderID,
		Metadata:        req.Metadata,
		TimeoutDuration: req.TimeoutDuration,
	}

	// Execute payment with circuit breaker protection
	var gatewayResp *payments.GatewayPaymentResponse
	var err error

	err = circuitBreaker.Execute(ctx, func() error {
		gateway, exists := eps.gatewayManager.GetGateway(req.Gateway)
		if !exists {
			return fmt.Errorf("gateway %s not available", req.Gateway)
		}

		var processErr error
		gatewayResp, processErr = gateway.ProcessPayment(ctx, gatewayReq)
		return processErr
	})

	processingTime := time.Since(startTime)
	attempt.ProcessingTimeMs = processingTime.Milliseconds()
	now := time.Now()
	attempt.CompletedAt = &now

	if err != nil {
		// Circuit breaker or other system error
		attempt.Success = false
		attempt.FailureType = payments.FailureTypeGatewayError
		attempt.FailureMessage = err.Error()
		attempt.GatewayResponse = map[string]interface{}{
			"error": err.Error(),
			"type":  "system_error",
		}

		eps.logger.Error("Payment attempt failed due to system error",
			"payment_id", payment.ID,
			"attempt", attemptNumber,
			"error", err)

		return attempt
	}

	if gatewayResp == nil {
		// Unexpected nil response
		attempt.Success = false
		attempt.FailureType = payments.FailureTypeInternalError
		attempt.FailureMessage = "Gateway returned nil response"
		attempt.GatewayResponse = map[string]interface{}{
			"error": "nil_response",
		}

		return attempt
	}

	// Process gateway response
	attempt.GatewayResponse = gatewayResp.GatewayResponse

	if gatewayResp.Status == "completed" {
		attempt.Success = true

		eps.logger.Debug("Payment attempt succeeded",
			"payment_id", payment.ID,
			"attempt", attemptNumber,
			"gateway_txn_id", gatewayResp.TransactionID,
			"processing_time_ms", processingTime.Milliseconds())
	} else {
		attempt.Success = false
		attempt.FailureType = gatewayResp.FailureType
		attempt.FailureMessage = gatewayResp.FailureMessage

		eps.logger.Warn("Payment attempt failed",
			"payment_id", payment.ID,
			"attempt", attemptNumber,
			"failure_type", gatewayResp.FailureType,
			"failure_message", gatewayResp.FailureMessage,
			"processing_time_ms", processingTime.Milliseconds())
	}

	return attempt
}

// scheduleAsyncRetry schedules a payment retry using the worker pool
func (eps *EnhancedPaymentService) scheduleAsyncRetry(ctx context.Context, req *payments.PaymentRequest, payment *models.Payment, retryAt time.Time) {
	// Create retry job
	job := &PaymentRetryJob{
		BaseJob:        workers.NewBaseJob(workers.JobTypePaymentRetry, workers.PriorityHigh, 3),
		PaymentID:      payment.ID,
		IdempotencyKey: req.IdempotencyKey,
		ScheduledAt:    retryAt,
		RetryPolicy:    req.RetryPolicy,
		executor:       eps,
	}

	// Schedule the job (for immediate scheduling, the worker pool will handle the delay)
	if err := eps.poolManager.SubmitJob(job); err != nil {
		eps.logger.Error("Failed to schedule payment retry job",
			"error", err,
			"payment_id", payment.ID,
			"retry_at", retryAt)
	} else {
		eps.logger.Info("Payment retry job scheduled",
			"payment_id", payment.ID,
			"retry_at", retryAt)
	}
}

// validatePaymentRequest validates the payment request
func (eps *EnhancedPaymentService) validatePaymentRequest(ctx context.Context, req *payments.PaymentRequest) error {
	if req.IdempotencyKey == "" {
		return fmt.Errorf("idempotency key is required")
	}
	if req.OrderID == "" {
		return fmt.Errorf("order ID is required")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if req.Currency == "" {
		return fmt.Errorf("currency is required")
	}
	if req.PaymentMethod == "" {
		return fmt.Errorf("payment method is required")
	}

	// Check if order exists and is in a payable state
	order, err := eps.orderRepo.GetByID(ctx, req.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}
	if order == nil {
		return fmt.Errorf("order not found")
	}

	// Validate payment amount against order total
	if req.Amount != order.TotalAmount {
		return fmt.Errorf("payment amount %.2f does not match order total %.2f", req.Amount, order.TotalAmount)
	}

	// Check order status
	if order.Status != models.OrderStatusPending && order.Status != models.OrderStatusConfirmed {
		return fmt.Errorf("order in status %s cannot be paid", order.Status)
	}

	// Check for existing successful payments for this order
	existingPayments, err := eps.paymentRepo.GetByOrderID(ctx, req.OrderID)
	if err != nil {
		return fmt.Errorf("failed to check existing payments: %w", err)
	}

	for _, payment := range existingPayments {
		if payment.IsCompleted() {
			return fmt.Errorf("order has already been paid")
		}
	}

	// Validate gateway availability
	if req.Gateway != "" {
		if _, exists := eps.gatewayManager.GetGateway(req.Gateway); !exists {
			return fmt.Errorf("gateway %s is not available", req.Gateway)
		}
	} else {
		// Select default gateway if not specified
		healthyGateways := eps.gatewayManager.GetHealthyGateways(ctx)
		if len(healthyGateways) == 0 {
			return fmt.Errorf("no healthy payment gateways available")
		}
		req.Gateway = healthyGateways[rand.Intn(len(healthyGateways))]
	}

	return nil
}

// createPaymentRecord creates a new payment record in the database
func (eps *EnhancedPaymentService) createPaymentRecord(ctx context.Context, req *payments.PaymentRequest) (*models.Payment, error) {
	// Serialize metadata
	var metadataJSON string
	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			eps.logger.Warn("Failed to serialize payment metadata", "error", err)
			metadataJSON = "{}"
		} else {
			metadataJSON = string(metadataBytes)
		}
	} else {
		metadataJSON = "{}"
	}

	// Set retry policy defaults
	maxRetries := 3
	if req.RetryPolicy != nil {
		maxRetries = req.RetryPolicy.MaxAttempts
	}

	payment := &models.Payment{
		OrderID:        req.OrderID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Status:         models.PaymentStatusPending,
		Method:         models.PaymentMethod(req.PaymentMethod),
		IdempotencyKey: req.IdempotencyKey,
		MaxRetries:     maxRetries,
		Gateway:        string(req.Gateway),
		Metadata:       metadataJSON,
	}

	if err := eps.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	eps.logger.Debug("Payment record created",
		"payment_id", payment.ID,
		"order_id", req.OrderID,
		"idempotency_key", req.IdempotencyKey)

	return payment, nil
}

// getPaymentResult retrieves a payment result from the database
func (eps *EnhancedPaymentService) getPaymentResult(ctx context.Context, paymentID string) (*payments.PaymentResult, error) {
	payment, err := eps.paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	if payment == nil {
		return nil, fmt.Errorf("payment not found")
	}

	result := &payments.PaymentResult{
		PaymentID:      payment.ID,
		IdempotencyKey: payment.IdempotencyKey,
		Status:         string(payment.Status),
		Success:        payment.IsCompleted(),
		Amount:         payment.Amount,
		Currency:       payment.Currency,
		AttemptCount:   payment.AttemptCount,
		CreatedAt:      payment.CreatedAt,
	}

	if payment.ProcessedAt != nil {
		result.CompletedAt = payment.ProcessedAt
		result.TotalProcessingTime = payment.ProcessedAt.Sub(payment.CreatedAt)
	}

	if payment.IsFailed() {
		result.FinalFailureMessage = payment.FailureReason
	}

	return result, nil
}

// Execute implements the JobExecutor interface for payment retry jobs
func (eps *EnhancedPaymentService) Execute(ctx context.Context, job workers.Job) error {
	retryJob, ok := job.(*PaymentRetryJob)
	if !ok {
		return fmt.Errorf("invalid job type for payment retry executor")
	}

	eps.logger.Info("Executing payment retry job",
		"payment_id", retryJob.PaymentID,
		"idempotency_key", retryJob.IdempotencyKey)

	// Wait until scheduled time if needed
	if time.Now().Before(retryJob.ScheduledAt) {
		waitTime := time.Until(retryJob.ScheduledAt)
		eps.logger.Debug("Waiting for scheduled retry time",
			"payment_id", retryJob.PaymentID,
			"wait_time_ms", waitTime.Milliseconds())
		time.Sleep(waitTime)
	}

	// Get payment record
	payment, err := eps.paymentRepo.GetByID(ctx, retryJob.PaymentID)
	if err != nil {
		return fmt.Errorf("failed to get payment for retry: %w", err)
	}
	if payment == nil {
		return fmt.Errorf("payment not found for retry: %s", retryJob.PaymentID)
	}

	// Check if payment is still retryable
	if !payment.IsRetryDue() {
		eps.logger.Info("Payment retry skipped - not due yet",
			"payment_id", retryJob.PaymentID,
			"next_retry_at", payment.NextRetryAt)
		return nil
	}

	// Reconstruct payment request from stored data
	var metadata map[string]interface{}
	if payment.Metadata != "" {
		if err := json.Unmarshal([]byte(payment.Metadata), &metadata); err != nil {
			eps.logger.Warn("Failed to deserialize payment metadata", "error", err, "payment_id", payment.ID)
			metadata = make(map[string]interface{})
		}
	}

	req := &payments.PaymentRequest{
		IdempotencyKey: payment.IdempotencyKey,
		OrderID:        payment.OrderID,
		Amount:         payment.Amount,
		Currency:       payment.Currency,
		PaymentMethod:  string(payment.Method),
		Gateway:        payments.PaymentGatewayType(payment.Gateway),
		Metadata:       metadata,
		RetryPolicy:    retryJob.RetryPolicy,
	}

	// Process the retry
	result := eps.processPaymentWithRetries(ctx, req, payment)

	// Update idempotency record
	status := "failed"
	if result.Success {
		status = "completed"
	}
	eps.idempotencyManager.UpdateIdempotencyRecord(ctx, req.IdempotencyKey, status, result)

	if result.Success {
		eps.logger.Info("Payment retry succeeded", "payment_id", retryJob.PaymentID)
	} else {
		eps.logger.Error("Payment retry failed",
			"payment_id", retryJob.PaymentID,
			"failure_type", result.FinalFailureType,
			"failure_message", result.FinalFailureMessage)
	}

	return nil
}

// PaymentRetryJob represents a payment retry job
type PaymentRetryJob struct {
	*workers.BaseJob
	PaymentID      string                  `json:"payment_id"`
	IdempotencyKey string                  `json:"idempotency_key"`
	ScheduledAt    time.Time               `json:"scheduled_at"`
	RetryPolicy    *payments.RetryPolicy   `json:"retry_policy"`
	executor       *EnhancedPaymentService `json:"-"`
}

// Execute implements the Job interface
func (prj *PaymentRetryJob) Execute(ctx context.Context) error {
	return prj.executor.Execute(ctx, prj)
}
