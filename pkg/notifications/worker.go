package notifications

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"easy-orders-backend/pkg/logger"
)

// DispatchWorker handles notification processing
type DispatchWorker struct {
	id         int
	dispatcher *NotificationDispatcher
	logger     *logger.Logger
	active     int32 // atomic flag for active status
}

// NewDispatchWorker creates a new dispatch worker
func NewDispatchWorker(id int, dispatcher *NotificationDispatcher, logger *logger.Logger) *DispatchWorker {
	return &DispatchWorker{
		id:         id,
		dispatcher: dispatcher,
		logger:     logger,
	}
}

// Start begins the worker's processing loop
func (dw *DispatchWorker) Start(ctx context.Context) {
	dw.logger.Debug("Dispatch worker started", "worker_id", dw.id)

	for {
		select {
		case <-ctx.Done():
			dw.logger.Debug("Dispatch worker stopped due to context cancellation", "worker_id", dw.id)
			return
		case <-dw.dispatcher.stopChan:
			dw.logger.Debug("Dispatch worker stopped due to dispatcher shutdown", "worker_id", dw.id)
			return
		default:
			// Worker is available, waiting for batch processing
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// ProcessBatch processes a batch of notifications for a specific channel
func (dw *DispatchWorker) ProcessBatch(channelName string, notifications []*Notification) {
	atomic.StoreInt32(&dw.active, 1)
	defer atomic.StoreInt32(&dw.active, 0)

	dw.logger.Debug("Worker processing notification batch",
		"worker_id", dw.id,
		"channel", channelName,
		"count", len(notifications))

	startTime := time.Now()

	// Get the notification channel
	channel, exists := dw.dispatcher.provider.GetChannel(channelName)
	if !exists {
		dw.logger.Error("Notification channel not found",
			"worker_id", dw.id,
			"channel", channelName)

		// Mark all notifications as failed
		for _, notification := range notifications {
			notification.MarkAsFailed(fmt.Errorf("channel %s not found", channelName))
			dw.queueForRetry(notification)
		}
		return
	}

	// Apply rate limiting based on channel limits
	rateLimit := channel.GetRateLimit()
	rateLimiter := time.NewTicker(time.Second / time.Duration(rateLimit))
	defer rateLimiter.Stop()

	successCount := 0
	failureCount := 0

	for _, notification := range notifications {
		// Wait for rate limit
		<-rateLimiter.C

		// Process single notification
		if dw.processNotification(channel, notification) {
			successCount++
		} else {
			failureCount++
		}
	}

	duration := time.Since(startTime)

	dw.logger.Info("Batch processing completed",
		"worker_id", dw.id,
		"channel", channelName,
		"total", len(notifications),
		"successful", successCount,
		"failed", failureCount,
		"duration_ms", duration.Milliseconds())

	// Update metrics
	dw.dispatcher.updateMetrics(func(m *DispatcherMetrics) {
		m.NotificationsSent += int64(successCount)
		m.NotificationsFailed += int64(failureCount)
		m.TotalLatency += duration
	})
}

// processNotification processes a single notification
func (dw *DispatchWorker) processNotification(channel NotificationChannel, notification *Notification) bool {
	dw.logger.Debug("Processing notification",
		"worker_id", dw.id,
		"notification_id", notification.ID,
		"type", notification.Type,
		"recipient", notification.Recipient)

	// Mark as sending
	notification.MarkAsSending()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Apply template if needed
	if err := dw.applyTemplate(notification); err != nil {
		dw.logger.Error("Failed to apply template",
			"worker_id", dw.id,
			"notification_id", notification.ID,
			"error", err)
		notification.MarkAsFailed(err)
		dw.queueForRetry(notification)
		return false
	}

	// Send notification
	startTime := time.Now()
	err := channel.Send(ctx, notification)
	duration := time.Since(startTime)

	if err != nil {
		dw.logger.Warn("Failed to send notification",
			"worker_id", dw.id,
			"notification_id", notification.ID,
			"channel", notification.Channel,
			"duration_ms", duration.Milliseconds(),
			"error", err)

		notification.MarkAsFailed(err)
		dw.queueForRetry(notification)
		return false
	}

	// Mark as sent
	notification.MarkAsSent()

	dw.logger.Debug("Notification sent successfully",
		"worker_id", dw.id,
		"notification_id", notification.ID,
		"channel", notification.Channel,
		"duration_ms", duration.Milliseconds())

	return true
}

// applyTemplate applies template to notification content
func (dw *DispatchWorker) applyTemplate(notification *Notification) error {
	if notification.TemplateID == "" {
		return nil // No template to apply
	}

	// Simple template application (in a real system, this would use a proper template engine)
	switch notification.Type {
	case NotificationTypeOrderConfirmation:
		notification.Subject = "Order Confirmation"
		notification.Body = dw.buildOrderConfirmationBody(notification.Data)
	case NotificationTypeOrderShipped:
		notification.Subject = "Order Shipped"
		notification.Body = dw.buildOrderShippedBody(notification.Data)
	case NotificationTypePaymentSuccess:
		notification.Subject = "Payment Successful"
		notification.Body = dw.buildPaymentSuccessBody(notification.Data)
	case NotificationTypeWelcome:
		notification.Subject = "Welcome!"
		notification.Body = dw.buildWelcomeBody(notification.Data)
	default:
		notification.Subject = "Notification"
		notification.Body = "You have a new notification"
	}

	return nil
}

// buildOrderConfirmationBody builds order confirmation message
func (dw *DispatchWorker) buildOrderConfirmationBody(data map[string]interface{}) string {
	orderID, _ := data["order_id"].(string)
	amount, _ := data["amount"].(float64)

	return fmt.Sprintf(
		"Thank you for your order!\n\nOrder ID: %s\nTotal Amount: $%.2f\n\nWe'll send you updates as your order progresses.",
		orderID, amount)
}

// buildOrderShippedBody builds order shipped message
func (dw *DispatchWorker) buildOrderShippedBody(data map[string]interface{}) string {
	orderID, _ := data["order_id"].(string)
	trackingNumber, _ := data["tracking_number"].(string)

	return fmt.Sprintf(
		"Great news! Your order has shipped.\n\nOrder ID: %s\nTracking Number: %s\n\nYou can track your package online.",
		orderID, trackingNumber)
}

// buildPaymentSuccessBody builds payment success message
func (dw *DispatchWorker) buildPaymentSuccessBody(data map[string]interface{}) string {
	amount, _ := data["amount"].(float64)
	method, _ := data["method"].(string)

	return fmt.Sprintf(
		"Your payment has been processed successfully.\n\nAmount: $%.2f\nPayment Method: %s\n\nThank you for your business!",
		amount, method)
}

// buildWelcomeBody builds welcome message
func (dw *DispatchWorker) buildWelcomeBody(data map[string]interface{}) string {
	name, _ := data["name"].(string)

	return fmt.Sprintf(
		"Welcome to our platform, %s!\n\nWe're excited to have you on board. Start exploring our products and enjoy your shopping experience.",
		name)
}

// queueForRetry queues a failed notification for retry
func (dw *DispatchWorker) queueForRetry(notification *Notification) {
	if !notification.ShouldRetry() {
		dw.logger.Warn("Notification exceeded max retries",
			"worker_id", dw.id,
			"notification_id", notification.ID,
			"retry_count", notification.RetryCount,
			"max_retries", notification.MaxRetries)
		return
	}

	select {
	case dw.dispatcher.retryQueue <- notification:
		dw.logger.Debug("Notification queued for retry",
			"worker_id", dw.id,
			"notification_id", notification.ID,
			"retry_count", notification.RetryCount)
	default:
		dw.logger.Error("Retry queue is full, dropping notification",
			"worker_id", dw.id,
			"notification_id", notification.ID)
	}
}

// IsActive returns whether the worker is currently processing notifications
func (dw *DispatchWorker) IsActive() bool {
	return atomic.LoadInt32(&dw.active) == 1
}

// GetID returns the worker's ID
func (dw *DispatchWorker) GetID() int {
	return dw.id
}
