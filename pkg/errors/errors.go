package errors

import (
	"fmt"
	"net/http"
)

// ErrorType represents different types of application errors
type ErrorType string

const (
	// Validation errors
	ErrorTypeValidation ErrorType = "VALIDATION_ERROR"
	ErrorTypeNotFound   ErrorType = "NOT_FOUND"
	ErrorTypeConflict   ErrorType = "CONFLICT"

	// Authentication/Authorization errors
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden    ErrorType = "FORBIDDEN"

	// Business logic errors
	ErrorTypeBusiness          ErrorType = "BUSINESS_ERROR"
	ErrorTypeInsufficientStock ErrorType = "INSUFFICIENT_STOCK"
	ErrorTypeInvalidTransition ErrorType = "INVALID_TRANSITION"
	ErrorTypePaymentFailed     ErrorType = "PAYMENT_FAILED"

	// Infrastructure errors
	ErrorTypeDatabase ErrorType = "DATABASE_ERROR"
	ErrorTypeExternal ErrorType = "EXTERNAL_SERVICE_ERROR"
	ErrorTypeInternal ErrorType = "INTERNAL_ERROR"

	// Rate limiting errors
	ErrorTypeRateLimit ErrorType = "RATE_LIMIT_EXCEEDED"
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

// Validation Errors
func NewValidationError(message string) *AppError {
	return NewAppError(ErrorTypeValidation, message, http.StatusBadRequest)
}

func NewValidationErrorWithDetails(message, details string) *AppError {
	err := NewValidationError(message)
	err.Details = details
	return err
}

// Not Found Errors
func NewNotFoundError(resource string) *AppError {
	return NewAppError(ErrorTypeNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

func NewNotFoundErrorWithID(resource, id string) *AppError {
	err := NewNotFoundError(resource)
	err.WithContext("id", id)
	return err
}

// Conflict Errors
func NewConflictError(message string) *AppError {
	return NewAppError(ErrorTypeConflict, message, http.StatusConflict)
}

func NewDuplicateError(resource, field, value string) *AppError {
	err := NewConflictError(fmt.Sprintf("%s with %s already exists", resource, field))
	err.WithContext("field", field).WithContext("value", value)
	return err
}

// Authentication/Authorization Errors
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

// Business Logic Errors
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

// Infrastructure Errors
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

// Rate Limiting Errors
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
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == errType
	}
	return false
}

// GetStatusCode extracts HTTP status code from error
func GetStatusCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}

// GetErrorResponse creates a standardized error response
func GetErrorResponse(err error) map[string]interface{} {
	if appErr, ok := err.(*AppError); ok {
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
