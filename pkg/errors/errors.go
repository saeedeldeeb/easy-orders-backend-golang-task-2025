package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorType represents different types of application errors
type ErrorType string

const (
	// ErrorTypeValidation Validation errors
	ErrorTypeValidation ErrorType = "VALIDATION_ERROR"
	ErrorTypeNotFound   ErrorType = "NOT_FOUND"
	ErrorTypeConflict   ErrorType = "CONFLICT"

	// ErrorTypeUnauthorized Authentication/Authorization errors
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden    ErrorType = "FORBIDDEN"

	// ErrorTypeBusiness Business logic errors
	ErrorTypeBusiness          ErrorType = "BUSINESS_ERROR"
	ErrorTypeInsufficientStock ErrorType = "INSUFFICIENT_STOCK"
	ErrorTypeInvalidTransition ErrorType = "INVALID_TRANSITION"
	ErrorTypePaymentFailed     ErrorType = "PAYMENT_FAILED"

	// ErrorTypeDatabase Infrastructure errors
	ErrorTypeDatabase ErrorType = "DATABASE_ERROR"
	ErrorTypeExternal ErrorType = "EXTERNAL_SERVICE_ERROR"
	ErrorTypeInternal ErrorType = "INTERNAL_ERROR"

	// ErrorTypeRateLimit Rate limiting errors
	ErrorTypeRateLimit ErrorType = "RATE_LIMIT_EXCEEDED"

	// ErrorTypeConcurrency Concurrency-related errors
	ErrorTypeConcurrencyConflict      ErrorType = "CONCURRENCY_CONFLICT"
	ErrorTypeOptimisticLockFailed     ErrorType = "OPTIMISTIC_LOCK_FAILED"
	ErrorTypeStockReservationConflict ErrorType = "STOCK_RESERVATION_CONFLICT"
	ErrorTypeLockTimeout              ErrorType = "LOCK_TIMEOUT"
)

// AppError represents a structured application error
type AppError struct {
	Type       ErrorType              `json:"type"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	StatusCode int                    `json:"-"`
	Cause      error                  `json:"-"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithCause adds the underlying cause
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// NewAppError creates a new application error
func NewAppError(errType ErrorType, message string, statusCode int) *AppError {
	return &AppError{
		Type:       errType,
		Message:    message,
		StatusCode: statusCode,
		Context:    make(map[string]interface{}),
	}
}

// NewValidationError Validation Errors
func NewValidationError(message string) *AppError {
	return NewAppError(ErrorTypeValidation, message, http.StatusBadRequest)
}

func NewValidationErrorWithDetails(message, details string) *AppError {
	err := NewValidationError(message)
	err.Details = details
	return err
}

// NewNotFoundError Not Found Errors
func NewNotFoundError(resource string) *AppError {
	return NewAppError(ErrorTypeNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

func NewNotFoundErrorWithID(resource, id string) *AppError {
	err := NewNotFoundError(resource)
	err.WithContext("id", id)
	return err
}

// NewConflictError Conflict Errors
func NewConflictError(message string) *AppError {
	return NewAppError(ErrorTypeConflict, message, http.StatusConflict)
}

func NewDuplicateError(resource, field, value string) *AppError {
	err := NewConflictError(fmt.Sprintf("%s with %s already exists", resource, field))
	err.WithContext("field", field).WithContext("value", value)
	return err
}

// NewUnauthorizedError Authentication/Authorization Errors
func NewUnauthorizedError(message string) *AppError {
	if message == "" {
		message = "Authentication required"
	}
	return NewAppError(ErrorTypeUnauthorized, message, http.StatusUnauthorized)
}

func NewForbiddenError(message string) *AppError {
	if message == "" {
		message = "Access forbidden"
	}
	return NewAppError(ErrorTypeForbidden, message, http.StatusForbidden)
}

// NewBusinessError Business Logic Errors
func NewBusinessError(message string) *AppError {
	return NewAppError(ErrorTypeBusiness, message, http.StatusBadRequest)
}

func NewInsufficientStockError(productID string, requested, available int) *AppError {
	err := NewAppError(ErrorTypeInsufficientStock, "Insufficient stock available", http.StatusConflict)
	err.WithContext("product_id", productID)
	err.WithContext("requested", requested)
	err.WithContext("available", available)
	return err
}

func NewInvalidTransitionError(from, to string) *AppError {
	err := NewAppError(ErrorTypeInvalidTransition, "Invalid status transition", http.StatusBadRequest)
	err.WithContext("from", from)
	err.WithContext("to", to)
	return err
}

func NewPaymentFailedError(reason string) *AppError {
	err := NewAppError(ErrorTypePaymentFailed, "Payment processing failed", http.StatusPaymentRequired)
	if reason != "" {
		err.Details = reason
	}
	return err
}

// NewDatabaseError Infrastructure Errors
func NewDatabaseError(message string, cause error) *AppError {
	err := NewAppError(ErrorTypeDatabase, message, http.StatusInternalServerError)
	err.WithCause(cause)
	return err
}

func NewExternalServiceError(service, message string, cause error) *AppError {
	err := NewAppError(ErrorTypeExternal, fmt.Sprintf("%s service error: %s", service, message), http.StatusBadGateway)
	err.WithContext("service", service)
	err.WithCause(cause)
	return err
}

func NewInternalError(message string, cause error) *AppError {
	err := NewAppError(ErrorTypeInternal, message, http.StatusInternalServerError)
	err.WithCause(cause)
	return err
}

// NewRateLimitError Rate Limiting Errors
func NewRateLimitError(limit int, window string) *AppError {
	err := NewAppError(ErrorTypeRateLimit, "Rate limit exceeded", http.StatusTooManyRequests)
	err.WithContext("limit", limit)
	err.WithContext("window", window)
	return err
}

// Helper functions for common error patterns

// WrapDatabaseError wraps a database error with context
func WrapDatabaseError(operation string, cause error) *AppError {
	return NewDatabaseError(fmt.Sprintf("Database operation failed: %s", operation), cause)
}

// WrapValidationError wraps a validation error with context
func WrapValidationError(field string, cause error) *AppError {
	err := NewValidationError(fmt.Sprintf("Validation failed for field: %s", field))
	err.WithContext("field", field)
	err.WithCause(cause)
	return err
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errType ErrorType) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == errType
	}
	return false
}

// GetStatusCode extracts HTTP status code from error
func GetStatusCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}

// GetErrorResponse creates a standardized error response
func GetErrorResponse(err error) map[string]interface{} {
	var appErr *AppError
	if errors.As(err, &appErr) {
		response := map[string]interface{}{
			"error": map[string]interface{}{
				"type":    appErr.Type,
				"message": appErr.Message,
			},
		}

		if appErr.Details != "" {
			response["error"].(map[string]interface{})["details"] = appErr.Details
		}

		if len(appErr.Context) > 0 {
			response["error"].(map[string]interface{})["context"] = appErr.Context
		}

		return response
	}

	// Fallback for non-AppError types
	return map[string]interface{}{
		"error": map[string]interface{}{
			"type":    ErrorTypeInternal,
			"message": "An unexpected error occurred",
		},
	}
}

// Concurrency Error Constructors

// NewConcurrencyConflictError creates a generic concurrency conflict error
func NewConcurrencyConflictError(message string) *AppError {
	return NewAppError(ErrorTypeConcurrencyConflict, message, http.StatusConflict)
}

// NewOptimisticLockError creates an optimistic locking failure error
func NewOptimisticLockError(resource, id string) *AppError {
	err := NewAppError(ErrorTypeOptimisticLockFailed,
		fmt.Sprintf("%s was modified by another process", resource),
		http.StatusConflict)
	err.WithContext("resource", resource)
	err.WithContext("id", id)
	err.Details = "The resource was updated by another request. Please retry your operation."
	return err
}

// NewStockReservationConflictError creates a stock reservation conflict error
func NewStockReservationConflictError(productID string, cause error) *AppError {
	err := NewAppError(ErrorTypeStockReservationConflict,
		"Stock reservation failed due to concurrent modification",
		http.StatusConflict)
	err.WithContext("product_id", productID)
	err.WithCause(cause)
	err.Details = "Another order was processed simultaneously. Please retry your order."
	return err
}

// NewLockTimeoutError creates a lock acquisition timeout error
func NewLockTimeoutError(resource, id string) *AppError {
	err := NewAppError(ErrorTypeLockTimeout,
		"Unable to acquire lock on resource",
		http.StatusConflict)
	err.WithContext("resource", resource)
	err.WithContext("id", id)
	err.Details = "The system is busy processing other requests. Please try again in a moment."
	return err
}

// IsConcurrencyError checks if an error is concurrency-related
func IsConcurrencyError(err error) bool {
	return IsErrorType(err, ErrorTypeConcurrencyConflict) ||
		IsErrorType(err, ErrorTypeOptimisticLockFailed) ||
		IsErrorType(err, ErrorTypeStockReservationConflict) ||
		IsErrorType(err, ErrorTypeLockTimeout)
}

// IsRetryableError checks if an error suggests the operation should be retried
func IsRetryableError(err error) bool {
	return IsConcurrencyError(err) || IsErrorType(err, ErrorTypeLockTimeout)
}
