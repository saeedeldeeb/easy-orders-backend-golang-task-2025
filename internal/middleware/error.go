package middleware

import (
	"net/http"

	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ErrorMiddleware provides centralized error handling
type ErrorMiddleware struct {
	logger *logger.Logger
}

// NewErrorMiddleware creates a new error handling middleware
func NewErrorMiddleware(logger *logger.Logger) *ErrorMiddleware {
	return &ErrorMiddleware{
		logger: logger,
	}
}

// Handler returns the error handling middleware
func (em *ErrorMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Execute the request
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			// Get the last error (most recent)
			err := c.Errors.Last().Err

			// Handle the error
			em.handleError(c, err)
			return
		}
	}
}

// handleError processes and responds to different types of errors
func (em *ErrorMiddleware) handleError(c *gin.Context, err error) {
	// Check if it's an AppError
	if appErr, ok := err.(*errors.AppError); ok {
		em.handleAppError(c, appErr)
		return
	}

	// Handle non-AppError types
	em.handleGenericError(c, err)
}

// handleAppError processes structured application errors
func (em *ErrorMiddleware) handleAppError(c *gin.Context, appErr *errors.AppError) {
	userID := getUserIDFromContext(c)

	// Log with appropriate level based on error type
	switch appErr.Type {
	case errors.ErrorTypeValidation, errors.ErrorTypeNotFound, errors.ErrorTypeConflict:
		em.logger.Warn("Client error", "error_type", appErr.Type, "message", appErr.Message, "status_code", appErr.StatusCode, "path", c.Request.URL.Path, "method", c.Request.Method, "user_id", userID)
	case errors.ErrorTypeUnauthorized, errors.ErrorTypeForbidden:
		em.logger.Warn("Authentication/Authorization error", "error_type", appErr.Type, "message", appErr.Message, "status_code", appErr.StatusCode, "path", c.Request.URL.Path, "method", c.Request.Method, "user_id", userID)
	case errors.ErrorTypeBusiness, errors.ErrorTypeInsufficientStock, errors.ErrorTypeInvalidTransition:
		em.logger.Info("Business logic error", "error_type", appErr.Type, "message", appErr.Message, "status_code", appErr.StatusCode, "path", c.Request.URL.Path, "method", c.Request.Method, "user_id", userID)
	case errors.ErrorTypePaymentFailed:
		em.logger.Error("Payment error", "error_type", appErr.Type, "message", appErr.Message, "status_code", appErr.StatusCode, "path", c.Request.URL.Path, "method", c.Request.Method, "user_id", userID)
	case errors.ErrorTypeDatabase, errors.ErrorTypeExternal, errors.ErrorTypeInternal:
		em.logger.Error("Infrastructure error", "error_type", appErr.Type, "message", appErr.Message, "status_code", appErr.StatusCode, "path", c.Request.URL.Path, "method", c.Request.Method, "user_id", userID)
	case errors.ErrorTypeRateLimit:
		em.logger.Warn("Rate limit exceeded", "error_type", appErr.Type, "message", appErr.Message, "status_code", appErr.StatusCode, "path", c.Request.URL.Path, "method", c.Request.Method, "user_id", userID)
	default:
		em.logger.Error("Unknown error type", "error_type", appErr.Type, "message", appErr.Message, "status_code", appErr.StatusCode, "path", c.Request.URL.Path, "method", c.Request.Method, "user_id", userID)
	}

	// Return error response
	response := errors.GetErrorResponse(appErr)
	c.JSON(appErr.StatusCode, response)
}

// handleGenericError processes non-structured errors
func (em *ErrorMiddleware) handleGenericError(c *gin.Context, err error) {
	em.logger.Error("Unhandled error",
		"error", err.Error(),
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"user_id", getUserIDFromContext(c),
	)

	// Create a generic internal error response
	internalErr := errors.NewInternalError("An unexpected error occurred", err)
	response := errors.GetErrorResponse(internalErr)
	c.JSON(http.StatusInternalServerError, response)
}

// getUserIDFromContext extracts user ID from context if available
func getUserIDFromContext(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			return userIDStr
		}
	}
	return "anonymous"
}

// AbortWithError is a helper function to abort with a structured error
func AbortWithError(c *gin.Context, err *errors.AppError) {
	c.Error(err)
	c.Abort()
}

// AbortWithValidationError is a helper for validation errors
func AbortWithValidationError(c *gin.Context, message, details string) {
	err := errors.NewValidationErrorWithDetails(message, details)
	AbortWithError(c, err)
}

// AbortWithNotFoundError is a helper for not found errors
func AbortWithNotFoundError(c *gin.Context, resource string) {
	err := errors.NewNotFoundError(resource)
	AbortWithError(c, err)
}

// AbortWithUnauthorizedError is a helper for unauthorized errors
func AbortWithUnauthorizedError(c *gin.Context, message string) {
	err := errors.NewUnauthorizedError(message)
	AbortWithError(c, err)
}

// AbortWithForbiddenError is a helper for forbidden errors
func AbortWithForbiddenError(c *gin.Context, message string) {
	err := errors.NewForbiddenError(message)
	AbortWithError(c, err)
}

// AbortWithInternalError is a helper for internal errors
func AbortWithInternalError(c *gin.Context, message string, cause error) {
	err := errors.NewInternalError(message, cause)
	AbortWithError(c, err)
}
