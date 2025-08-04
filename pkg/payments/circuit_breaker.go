package payments

import (
	"context"
	"fmt"
	"sync"
	"time"

	"easy-orders-backend/pkg/logger"
)

// CircuitBreaker implements the circuit breaker pattern for payment gateways
type CircuitBreaker struct {
	config          *CircuitBreakerConfig
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	lastStateChange time.Time
	mutex           sync.RWMutex
	logger          *logger.Logger
	gatewayName     string
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(gatewayName string, config *CircuitBreakerConfig, logger *logger.Logger) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	return &CircuitBreaker{
		config:          config,
		state:           CircuitBreakerClosed,
		lastStateChange: time.Now(),
		logger:          logger,
		gatewayName:     gatewayName,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	// Check if we can execute
	if !cb.CanExecute() {
		return fmt.Errorf("circuit breaker is open for gateway %s", cb.gatewayName)
	}

	// Execute the operation
	startTime := time.Now()
	err := operation()
	duration := time.Since(startTime)

	// Record the result
	if err != nil {
		cb.RecordFailure(err)
		cb.logger.Warn("Circuit breaker recorded failure",
			"gateway", cb.gatewayName,
			"duration_ms", duration.Milliseconds(),
			"error", err)
	} else {
		cb.RecordSuccess()
		cb.logger.Debug("Circuit breaker recorded success",
			"gateway", cb.gatewayName,
			"duration_ms", duration.Milliseconds())
	}

	return err
}

// CanExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastStateChange) >= cb.config.ResetTimeout {
			cb.transitionToHalfOpen()
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.successCount++

	switch cb.state {
	case CircuitBreakerHalfOpen:
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.transitionToClosed()
		}
	case CircuitBreakerOpen:
		// This shouldn't happen, but reset if it does
		cb.transitionToClosed()
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitBreakerClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.transitionToOpen()
		}
	case CircuitBreakerHalfOpen:
		cb.transitionToOpen()
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"gateway":               cb.gatewayName,
		"state":                 string(cb.state),
		"failure_count":         cb.failureCount,
		"success_count":         cb.successCount,
		"last_failure_time":     cb.lastFailureTime,
		"last_state_change":     cb.lastStateChange,
		"failure_threshold":     cb.config.FailureThreshold,
		"success_threshold":     cb.config.SuccessThreshold,
		"timeout_seconds":       cb.config.Timeout.Seconds(),
		"reset_timeout_seconds": cb.config.ResetTimeout.Seconds(),
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.transitionToClosed()
	cb.logger.Info("Circuit breaker manually reset", "gateway", cb.gatewayName)
}

// transitionToClosed transitions the circuit breaker to closed state
func (cb *CircuitBreaker) transitionToClosed() {
	previousState := cb.state
	cb.state = CircuitBreakerClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastStateChange = time.Now()

	if previousState != CircuitBreakerClosed {
		cb.logger.Info("Circuit breaker transitioned to CLOSED",
			"gateway", cb.gatewayName,
			"previous_state", string(previousState))
	}
}

// transitionToOpen transitions the circuit breaker to open state
func (cb *CircuitBreaker) transitionToOpen() {
	previousState := cb.state
	cb.state = CircuitBreakerOpen
	cb.successCount = 0
	cb.lastStateChange = time.Now()

	if previousState != CircuitBreakerOpen {
		cb.logger.Warn("Circuit breaker transitioned to OPEN",
			"gateway", cb.gatewayName,
			"previous_state", string(previousState),
			"failure_count", cb.failureCount)
	}
}

// transitionToHalfOpen transitions the circuit breaker to half-open state
func (cb *CircuitBreaker) transitionToHalfOpen() {
	previousState := cb.state
	cb.state = CircuitBreakerHalfOpen
	cb.successCount = 0
	cb.lastStateChange = time.Now()

	if previousState != CircuitBreakerHalfOpen {
		cb.logger.Info("Circuit breaker transitioned to HALF-OPEN",
			"gateway", cb.gatewayName,
			"previous_state", string(previousState))
	}
}

// CircuitBreakerManager manages multiple circuit breakers for different gateways
type CircuitBreakerManager struct {
	breakers map[PaymentGatewayType]*CircuitBreaker
	mutex    sync.RWMutex
	logger   *logger.Logger
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(logger *logger.Logger) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[PaymentGatewayType]*CircuitBreaker),
		logger:   logger,
	}
}

// GetCircuitBreaker gets or creates a circuit breaker for a gateway
func (cbm *CircuitBreakerManager) GetCircuitBreaker(gateway PaymentGatewayType) *CircuitBreaker {
	cbm.mutex.RLock()
	breaker, exists := cbm.breakers[gateway]
	cbm.mutex.RUnlock()

	if exists {
		return breaker
	}

	// Create new circuit breaker
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := cbm.breakers[gateway]; exists {
		return breaker
	}

	breaker = NewCircuitBreaker(string(gateway), DefaultCircuitBreakerConfig(), cbm.logger)
	cbm.breakers[gateway] = breaker

	cbm.logger.Info("Created circuit breaker for gateway", "gateway", string(gateway))

	return breaker
}

// GetAllStats returns statistics for all circuit breakers
func (cbm *CircuitBreakerManager) GetAllStats() map[string]interface{} {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	stats := make(map[string]interface{})
	for gateway, breaker := range cbm.breakers {
		stats[string(gateway)] = breaker.GetStats()
	}

	return stats
}

// ResetAll resets all circuit breakers
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	for _, breaker := range cbm.breakers {
		breaker.Reset()
	}

	cbm.logger.Info("All circuit breakers reset")
}
