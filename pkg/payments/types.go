package payments

import (
	"time"
)

// PaymentFailureType categorizes different types of payment failures
type PaymentFailureType string

const (
	// Retriable failures - can be retried with same parameters
	FailureTypeNetworkError     PaymentFailureType = "network_error"
	FailureTypeGatewayTimeout   PaymentFailureType = "gateway_timeout"
	FailureTypeGatewayError     PaymentFailureType = "gateway_error"
	FailureTypeRateLimited      PaymentFailureType = "rate_limited"
	FailureTypeTemporaryDecline PaymentFailureType = "temporary_decline"

	// Non-retriable failures - require user intervention
	FailureTypeInsufficientFunds PaymentFailureType = "insufficient_funds"
	FailureTypeInvalidCard       PaymentFailureType = "invalid_card"
	FailureTypeExpiredCard       PaymentFailureType = "expired_card"
	FailureTypeFraudSuspected    PaymentFailureType = "fraud_suspected"
	FailureTypeCardBlocked       PaymentFailureType = "card_blocked"
	FailureTypeInvalidAmount     PaymentFailureType = "invalid_amount"
	FailureTypeInvalidCurrency   PaymentFailureType = "invalid_currency"

	// System failures - require investigation
	FailureTypeConfigurationError  PaymentFailureType = "configuration_error"
	FailureTypeInternalError       PaymentFailureType = "internal_error"
	FailureTypeAuthenticationError PaymentFailureType = "authentication_error"
)

// PaymentGatewayType defines different payment gateways
type PaymentGatewayType string

const (
	GatewayTypeStripe    PaymentGatewayType = "stripe"
	GatewayTypePayPal    PaymentGatewayType = "paypal"
	GatewayTypeSquare    PaymentGatewayType = "square"
	GatewayTypeBraintree PaymentGatewayType = "braintree"
	GatewayTypeMock      PaymentGatewayType = "mock" // For testing
)

// PaymentAttempt represents a single payment processing attempt
type PaymentAttempt struct {
	AttemptNumber    int                    `json:"attempt_number"`
	Gateway          PaymentGatewayType     `json:"gateway"`
	StartedAt        time.Time              `json:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	Success          bool                   `json:"success"`
	FailureType      PaymentFailureType     `json:"failure_type,omitempty"`
	FailureMessage   string                 `json:"failure_message,omitempty"`
	GatewayResponse  map[string]interface{} `json:"gateway_response,omitempty"`
	ProcessingTimeMs int64                  `json:"processing_time_ms"`
}

// RetryPolicy defines how payment retries should be handled
type RetryPolicy struct {
	MaxAttempts       int                  `json:"max_attempts"`
	InitialDelay      time.Duration        `json:"initial_delay"`
	MaxDelay          time.Duration        `json:"max_delay"`
	BackoffMultiplier float64              `json:"backoff_multiplier"`
	JitterPercent     float64              `json:"jitter_percent"`
	RetriableFailures []PaymentFailureType `json:"retriable_failures"`
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:       5,
		InitialDelay:      time.Second,
		MaxDelay:          5 * time.Minute,
		BackoffMultiplier: 2.0,
		JitterPercent:     0.1,
		RetriableFailures: []PaymentFailureType{
			FailureTypeNetworkError,
			FailureTypeGatewayTimeout,
			FailureTypeGatewayError,
			FailureTypeRateLimited,
			FailureTypeTemporaryDecline,
		},
	}
}

// AggressiveRetryPolicy returns a more aggressive retry policy for critical payments
func AggressiveRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:       8,
		InitialDelay:      500 * time.Millisecond,
		MaxDelay:          2 * time.Minute,
		BackoffMultiplier: 1.5,
		JitterPercent:     0.15,
		RetriableFailures: []PaymentFailureType{
			FailureTypeNetworkError,
			FailureTypeGatewayTimeout,
			FailureTypeGatewayError,
			FailureTypeRateLimited,
			FailureTypeTemporaryDecline,
		},
	}
}

// ConservativeRetryPolicy returns a conservative retry policy for low-priority payments
func ConservativeRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:       3,
		InitialDelay:      2 * time.Second,
		MaxDelay:          10 * time.Minute,
		BackoffMultiplier: 3.0,
		JitterPercent:     0.05,
		RetriableFailures: []PaymentFailureType{
			FailureTypeNetworkError,
			FailureTypeGatewayTimeout,
		},
	}
}

// IsRetriable checks if a failure type is retriable according to the policy
func (p *RetryPolicy) IsRetriable(failureType PaymentFailureType) bool {
	for _, retriableType := range p.RetriableFailures {
		if retriableType == failureType {
			return true
		}
	}
	return false
}

// CalculateNextRetryDelay calculates the delay before the next retry attempt
func (p *RetryPolicy) CalculateNextRetryDelay(attemptNumber int) time.Duration {
	if attemptNumber >= p.MaxAttempts {
		return 0 // No more retries
	}

	// Calculate exponential backoff
	delay := float64(p.InitialDelay)
	for i := 1; i < attemptNumber; i++ {
		delay *= p.BackoffMultiplier
	}

	// Apply maximum delay cap
	if time.Duration(delay) > p.MaxDelay {
		delay = float64(p.MaxDelay)
	}

	// Apply jitter to prevent thundering herd
	jitter := delay * p.JitterPercent
	if jitter > 0 {
		// Random jitter between -jitter and +jitter
		jitterRange := jitter * 2
		jitterOffset := (float64(time.Now().UnixNano()%1000)/1000.0)*jitterRange - jitter
		delay += jitterOffset
	}

	// Ensure minimum delay
	if delay < float64(p.InitialDelay) {
		delay = float64(p.InitialDelay)
	}

	return time.Duration(delay)
}

// PaymentRequest represents a payment processing request with idempotency
type PaymentRequest struct {
	IdempotencyKey  string                 `json:"idempotency_key" validate:"required"`
	OrderID         string                 `json:"order_id" validate:"required"`
	Amount          float64                `json:"amount" validate:"required,gt=0"`
	Currency        string                 `json:"currency" validate:"required,len=3"`
	PaymentMethod   string                 `json:"payment_method" validate:"required"`
	Gateway         PaymentGatewayType     `json:"gateway"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	RetryPolicy     *RetryPolicy           `json:"retry_policy,omitempty"`
	CallbackURL     string                 `json:"callback_url,omitempty"`
	TimeoutDuration time.Duration          `json:"timeout_duration,omitempty"`
}

// PaymentResult represents the complete result of a payment processing attempt
type PaymentResult struct {
	PaymentID           string             `json:"payment_id"`
	IdempotencyKey      string             `json:"idempotency_key"`
	Status              string             `json:"status"`
	Success             bool               `json:"success"`
	Amount              float64            `json:"amount"`
	Currency            string             `json:"currency"`
	AttemptCount        int                `json:"attempt_count"`
	TotalProcessingTime time.Duration      `json:"total_processing_time"`
	Attempts            []PaymentAttempt   `json:"attempts"`
	FinalFailureType    PaymentFailureType `json:"final_failure_type,omitempty"`
	FinalFailureMessage string             `json:"final_failure_message,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	CompletedAt         *time.Time         `json:"completed_at,omitempty"`
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"    // Normal operation
	CircuitBreakerOpen     CircuitBreakerState = "open"      // Failing fast
	CircuitBreakerHalfOpen CircuitBreakerState = "half_open" // Testing recovery
)

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold"` // Number of failures to open circuit
	SuccessThreshold int           `json:"success_threshold"` // Number of successes to close circuit
	Timeout          time.Duration `json:"timeout"`           // How long circuit stays open
	ResetTimeout     time.Duration `json:"reset_timeout"`     // How long to wait before allowing test requests
}

// DefaultCircuitBreakerConfig returns a sensible default circuit breaker configuration
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		ResetTimeout:     60 * time.Second,
	}
}

// PaymentMetrics tracks payment processing metrics
type PaymentMetrics struct {
	TotalPayments         int64                                  `json:"total_payments"`
	SuccessfulPayments    int64                                  `json:"successful_payments"`
	FailedPayments        int64                                  `json:"failed_payments"`
	RetriedPayments       int64                                  `json:"retried_payments"`
	AverageProcessingTime time.Duration                          `json:"average_processing_time"`
	SuccessRate           float64                                `json:"success_rate"`
	GatewayMetrics        map[PaymentGatewayType]*GatewayMetrics `json:"gateway_metrics"`
}

// GatewayMetrics tracks metrics per payment gateway
type GatewayMetrics struct {
	TotalRequests       int64               `json:"total_requests"`
	SuccessfulRequests  int64               `json:"successful_requests"`
	FailedRequests      int64               `json:"failed_requests"`
	AverageLatency      time.Duration       `json:"average_latency"`
	CircuitBreakerState CircuitBreakerState `json:"circuit_breaker_state"`
	LastFailureTime     *time.Time          `json:"last_failure_time,omitempty"`
}
