package payments

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"easy-orders-backend/pkg/logger"
)

// PaymentGateway defines the interface for payment gateway implementations
type PaymentGateway interface {
	// ProcessPayment processes a payment request
	ProcessPayment(ctx context.Context, req *GatewayPaymentRequest) (*GatewayPaymentResponse, error)

	// RefundPayment processes a refund request
	RefundPayment(ctx context.Context, req *GatewayRefundRequest) (*GatewayRefundResponse, error)

	// GetPaymentStatus retrieves the status of a payment
	GetPaymentStatus(ctx context.Context, gatewayTransactionID string) (*GatewayPaymentStatus, error)

	// GetGatewayType returns the gateway type
	GetGatewayType() PaymentGatewayType

	// IsHealthy checks if the gateway is healthy
	IsHealthy(ctx context.Context) bool
}

// GatewayPaymentRequest represents a payment request to a gateway
type GatewayPaymentRequest struct {
	Amount          float64                `json:"amount"`
	Currency        string                 `json:"currency"`
	PaymentMethod   string                 `json:"payment_method"`
	IdempotencyKey  string                 `json:"idempotency_key"`
	OrderReference  string                 `json:"order_reference"`
	CustomerEmail   string                 `json:"customer_email,omitempty"`
	CustomerName    string                 `json:"customer_name,omitempty"`
	BillingAddress  *BillingAddress        `json:"billing_address,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	TimeoutDuration time.Duration          `json:"timeout_duration,omitempty"`
}

// GatewayPaymentResponse represents a payment response from a gateway
type GatewayPaymentResponse struct {
	TransactionID     string                 `json:"transaction_id"`
	Status            string                 `json:"status"`
	Amount            float64                `json:"amount"`
	Currency          string                 `json:"currency"`
	ProcessingFee     float64                `json:"processing_fee,omitempty"`
	GatewayResponse   map[string]interface{} `json:"gateway_response,omitempty"`
	AuthorizationCode string                 `json:"authorization_code,omitempty"`
	ProcessingTimeMs  int64                  `json:"processing_time_ms"`
	FailureType       PaymentFailureType     `json:"failure_type,omitempty"`
	FailureMessage    string                 `json:"failure_message,omitempty"`
}

// GatewayRefundRequest represents a refund request to a gateway
type GatewayRefundRequest struct {
	OriginalTransactionID string                 `json:"original_transaction_id"`
	Amount                float64                `json:"amount"`
	Currency              string                 `json:"currency"`
	Reason                string                 `json:"reason,omitempty"`
	IdempotencyKey        string                 `json:"idempotency_key"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// GatewayRefundResponse represents a refund response from a gateway
type GatewayRefundResponse struct {
	RefundID         string                 `json:"refund_id"`
	Status           string                 `json:"status"`
	Amount           float64                `json:"amount"`
	Currency         string                 `json:"currency"`
	ProcessingFee    float64                `json:"processing_fee,omitempty"`
	GatewayResponse  map[string]interface{} `json:"gateway_response,omitempty"`
	ProcessingTimeMs int64                  `json:"processing_time_ms"`
	FailureType      PaymentFailureType     `json:"failure_type,omitempty"`
	FailureMessage   string                 `json:"failure_message,omitempty"`
}

// GatewayPaymentStatus represents the status of a payment from a gateway
type GatewayPaymentStatus struct {
	TransactionID   string                 `json:"transaction_id"`
	Status          string                 `json:"status"`
	Amount          float64                `json:"amount"`
	Currency        string                 `json:"currency"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	GatewayResponse map[string]interface{} `json:"gateway_response,omitempty"`
	FailureType     PaymentFailureType     `json:"failure_type,omitempty"`
	FailureMessage  string                 `json:"failure_message,omitempty"`
}

// BillingAddress represents a billing address
type BillingAddress struct {
	Name       string `json:"name"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// MockPaymentGateway implements a mock payment gateway for testing
type MockPaymentGateway struct {
	gatewayType  PaymentGatewayType
	logger       *logger.Logger
	failureRate  float64       // Percentage of requests that should fail (0.0 - 1.0)
	latencyRange time.Duration // Maximum latency to simulate
}

// NewMockPaymentGateway creates a new mock payment gateway
func NewMockPaymentGateway(gatewayType PaymentGatewayType, failureRate float64, maxLatency time.Duration, logger *logger.Logger) *MockPaymentGateway {
	return &MockPaymentGateway{
		gatewayType:  gatewayType,
		logger:       logger,
		failureRate:  failureRate,
		latencyRange: maxLatency,
	}
}

// ProcessPayment simulates payment processing
func (mpg *MockPaymentGateway) ProcessPayment(ctx context.Context, req *GatewayPaymentRequest) (*GatewayPaymentResponse, error) {
	startTime := time.Now()

	// Simulate processing latency
	latency := time.Duration(rand.Int63n(int64(mpg.latencyRange)))
	time.Sleep(latency)

	processingTime := time.Since(startTime)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("payment processing cancelled: %w", ctx.Err())
	default:
	}

	// Simulate different failure scenarios
	if rand.Float64() < mpg.failureRate {
		return mpg.simulateFailure(req, processingTime)
	}

	// Simulate successful payment
	return &GatewayPaymentResponse{
		TransactionID:     fmt.Sprintf("%s_%d", mpg.gatewayType, time.Now().UnixNano()),
		Status:            "completed",
		Amount:            req.Amount,
		Currency:          req.Currency,
		ProcessingFee:     req.Amount * 0.029, // 2.9% processing fee
		AuthorizationCode: fmt.Sprintf("AUTH_%d", rand.Int63n(999999)),
		ProcessingTimeMs:  processingTime.Milliseconds(),
		GatewayResponse: map[string]interface{}{
			"gateway":         string(mpg.gatewayType),
			"processed_at":    time.Now().UTC(),
			"reference":       req.OrderReference,
			"idempotency_key": req.IdempotencyKey,
		},
	}, nil
}

// RefundPayment simulates refund processing
func (mpg *MockPaymentGateway) RefundPayment(ctx context.Context, req *GatewayRefundRequest) (*GatewayRefundResponse, error) {
	startTime := time.Now()

	// Simulate processing latency (refunds are usually faster)
	latency := time.Duration(rand.Int63n(int64(mpg.latencyRange / 2)))
	time.Sleep(latency)

	processingTime := time.Since(startTime)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("refund processing cancelled: %w", ctx.Err())
	default:
	}

	// Simulate lower failure rate for refunds (usually more reliable)
	if rand.Float64() < (mpg.failureRate * 0.3) {
		return &GatewayRefundResponse{
			RefundID:         fmt.Sprintf("REF_%s_%d", mpg.gatewayType, time.Now().UnixNano()),
			Status:           "failed",
			Amount:           req.Amount,
			Currency:         req.Currency,
			ProcessingTimeMs: processingTime.Milliseconds(),
			FailureType:      FailureTypeGatewayError,
			FailureMessage:   "Simulated refund failure",
			GatewayResponse: map[string]interface{}{
				"gateway":    string(mpg.gatewayType),
				"failed_at":  time.Now().UTC(),
				"error_code": "REFUND_FAILED",
			},
		}, nil
	}

	// Simulate successful refund
	return &GatewayRefundResponse{
		RefundID:         fmt.Sprintf("REF_%s_%d", mpg.gatewayType, time.Now().UnixNano()),
		Status:           "completed",
		Amount:           req.Amount,
		Currency:         req.Currency,
		ProcessingFee:    req.Amount * 0.029, // Same processing fee
		ProcessingTimeMs: processingTime.Milliseconds(),
		GatewayResponse: map[string]interface{}{
			"gateway":         string(mpg.gatewayType),
			"processed_at":    time.Now().UTC(),
			"original_txn":    req.OriginalTransactionID,
			"idempotency_key": req.IdempotencyKey,
		},
	}, nil
}

// GetPaymentStatus simulates payment status retrieval
func (mpg *MockPaymentGateway) GetPaymentStatus(ctx context.Context, gatewayTransactionID string) (*GatewayPaymentStatus, error) {
	// Simulate API call latency
	time.Sleep(time.Duration(rand.Int63n(int64(100 * time.Millisecond))))

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("status check cancelled: %w", ctx.Err())
	default:
	}

	// Simulate different statuses
	statuses := []string{"pending", "processing", "completed", "failed"}
	status := statuses[rand.Intn(len(statuses))]

	now := time.Now()
	return &GatewayPaymentStatus{
		TransactionID: gatewayTransactionID,
		Status:        status,
		Amount:        100.00, // Mock amount
		Currency:      "USD",
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now,
		GatewayResponse: map[string]interface{}{
			"gateway":        string(mpg.gatewayType),
			"last_checked":   now.UTC(),
			"transaction_id": gatewayTransactionID,
		},
	}, nil
}

// GetGatewayType returns the gateway type
func (mpg *MockPaymentGateway) GetGatewayType() PaymentGatewayType {
	return mpg.gatewayType
}

// IsHealthy simulates health check
func (mpg *MockPaymentGateway) IsHealthy(ctx context.Context) bool {
	// Simulate occasional downtime
	return rand.Float64() > 0.05 // 95% uptime
}

// simulateFailure simulates different types of payment failures
func (mpg *MockPaymentGateway) simulateFailure(req *GatewayPaymentRequest, processingTime time.Duration) (*GatewayPaymentResponse, error) {
	failureScenarios := []struct {
		failureType PaymentFailureType
		message     string
		weight      int
	}{
		{FailureTypeInsufficientFunds, "Insufficient funds", 30},
		{FailureTypeInvalidCard, "Invalid card number", 15},
		{FailureTypeExpiredCard, "Card expired", 10},
		{FailureTypeNetworkError, "Network timeout", 20},
		{FailureTypeGatewayTimeout, "Gateway timeout", 15},
		{FailureTypeRateLimited, "Rate limit exceeded", 5},
		{FailureTypeFraudSuspected, "Transaction flagged for fraud", 3},
		{FailureTypeCardBlocked, "Card blocked by issuer", 2},
	}

	// Weighted random selection
	totalWeight := 0
	for _, scenario := range failureScenarios {
		totalWeight += scenario.weight
	}

	randomValue := rand.Intn(totalWeight)
	currentWeight := 0

	for _, scenario := range failureScenarios {
		currentWeight += scenario.weight
		if randomValue < currentWeight {
			return &GatewayPaymentResponse{
				TransactionID:    fmt.Sprintf("FAILED_%s_%d", mpg.gatewayType, time.Now().UnixNano()),
				Status:           "failed",
				Amount:           req.Amount,
				Currency:         req.Currency,
				ProcessingTimeMs: processingTime.Milliseconds(),
				FailureType:      scenario.failureType,
				FailureMessage:   scenario.message,
				GatewayResponse: map[string]interface{}{
					"gateway":         string(mpg.gatewayType),
					"failed_at":       time.Now().UTC(),
					"error_code":      string(scenario.failureType),
					"reference":       req.OrderReference,
					"idempotency_key": req.IdempotencyKey,
				},
			}, nil
		}
	}

	// Fallback to generic failure
	return &GatewayPaymentResponse{
		TransactionID:    fmt.Sprintf("FAILED_%s_%d", mpg.gatewayType, time.Now().UnixNano()),
		Status:           "failed",
		Amount:           req.Amount,
		Currency:         req.Currency,
		ProcessingTimeMs: processingTime.Milliseconds(),
		FailureType:      FailureTypeInternalError,
		FailureMessage:   "Internal gateway error",
		GatewayResponse: map[string]interface{}{
			"gateway":    string(mpg.gatewayType),
			"failed_at":  time.Now().UTC(),
			"error_code": "INTERNAL_ERROR",
		},
	}, nil
}

// PaymentGatewayManager manages multiple payment gateways
type PaymentGatewayManager struct {
	gateways map[PaymentGatewayType]PaymentGateway
	logger   *logger.Logger
}

// NewPaymentGatewayManager creates a new payment gateway manager
func NewPaymentGatewayManager(logger *logger.Logger) *PaymentGatewayManager {
	return &PaymentGatewayManager{
		gateways: make(map[PaymentGatewayType]PaymentGateway),
		logger:   logger,
	}
}

// RegisterGateway registers a payment gateway
func (pgm *PaymentGatewayManager) RegisterGateway(gateway PaymentGateway) {
	pgm.gateways[gateway.GetGatewayType()] = gateway
	pgm.logger.Info("Payment gateway registered", "type", string(gateway.GetGatewayType()))
}

// GetGateway retrieves a payment gateway by type
func (pgm *PaymentGatewayManager) GetGateway(gatewayType PaymentGatewayType) (PaymentGateway, bool) {
	gateway, exists := pgm.gateways[gatewayType]
	return gateway, exists
}

// GetAvailableGateways returns all available gateway types
func (pgm *PaymentGatewayManager) GetAvailableGateways() []PaymentGatewayType {
	var gateways []PaymentGatewayType
	for gatewayType := range pgm.gateways {
		gateways = append(gateways, gatewayType)
	}
	return gateways
}

// GetHealthyGateways returns all healthy gateway types
func (pgm *PaymentGatewayManager) GetHealthyGateways(ctx context.Context) []PaymentGatewayType {
	var healthyGateways []PaymentGatewayType
	for gatewayType, gateway := range pgm.gateways {
		if gateway.IsHealthy(ctx) {
			healthyGateways = append(healthyGateways, gatewayType)
		}
	}
	return healthyGateways
}
