package concurrency

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"easy-orders-backend/pkg/logger"
)

// RetryConfig defines retry behavior for concurrent operations
type RetryConfig struct {
	MaxAttempts     int           `json:"max_attempts"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	JitterPercent   float64       `json:"jitter_percent"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.1,
		RetryableErrors: []string{
			"conflict",
			"version mismatch",
			"optimistic lock",
			"concurrent modification",
			"deadlock",
		},
	}
}

// RetryableFunc is a function that can be retried
type RetryableFunc func() error

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(ctx context.Context, config *RetryConfig, operation RetryableFunc, logger *logger.Logger) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		default:
		}

		err := operation()
		if err == nil {
			if attempt > 1 {
				logger.Info("Operation succeeded after retry",
					"attempt", attempt,
					"total_attempts", config.MaxAttempts)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err, config.RetryableErrors) {
			logger.Debug("Non-retryable error encountered", "error", err, "attempt", attempt)
			return err
		}

		// Don't wait if this was the last attempt
		if attempt == config.MaxAttempts {
			break
		}

		// Calculate delay with exponential backoff and jitter
		delay := calculateDelay(attempt, config)

		logger.Debug("Retrying operation after error",
			"error", err,
			"attempt", attempt,
			"max_attempts", config.MaxAttempts,
			"delay_ms", delay.Milliseconds())

		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled during backoff: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	logger.Warn("Operation failed after all retry attempts",
		"final_error", lastErr,
		"attempts", config.MaxAttempts)
	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// isRetryableError checks if an error should trigger a retry
func isRetryableError(err error, retryableErrors []string) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	for _, retryableErr := range retryableErrors {
		if strings.Contains(errStr, strings.ToLower(retryableErr)) {
			return true
		}
	}
	return false
}

// calculateDelay calculates the delay for the given attempt with jitter
func calculateDelay(attempt int, config *RetryConfig) time.Duration {
	// Exponential backoff: delay = initial_delay * (backoff_factor ^ (attempt-1))
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1))

	// Apply max delay limit
	if time.Duration(delay) > config.MaxDelay {
		delay = float64(config.MaxDelay)
	}

	// Add jitter to avoid thundering herd
	if config.JitterPercent > 0 {
		jitter := delay * config.JitterPercent * (rand.Float64() - 0.5) * 2
		delay += jitter
	}

	// Ensure delay is not negative
	if delay < 0 {
		delay = float64(config.InitialDelay)
	}

	return time.Duration(delay)
}

// ConcurrentRetry executes multiple operations concurrently with retry logic
func ConcurrentRetry(ctx context.Context, config *RetryConfig, operations []RetryableFunc, logger *logger.Logger) error {
	if len(operations) == 0 {
		return nil
	}

	type result struct {
		index int
		err   error
	}

	resultChan := make(chan result, len(operations))

	// Start all operations concurrently
	for i, op := range operations {
		go func(index int, operation RetryableFunc) {
			err := RetryWithBackoff(ctx, config, operation, logger)
			resultChan <- result{index: index, err: err}
		}(i, op)
	}

	// Collect results
	var errors []error
	for i := 0; i < len(operations); i++ {
		res := <-resultChan
		if res.err != nil {
			errors = append(errors, fmt.Errorf("operation %d failed: %w", res.index, res.err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("concurrent operations failed: %v", errors)
	}

	return nil
}
