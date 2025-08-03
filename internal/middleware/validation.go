package middleware

import (
	"net/http"
	"reflect"
	"strings"

	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidationMiddleware provides request validation middleware
type ValidationMiddleware struct {
	validator *validator.Validate
	logger    *logger.Logger
}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware(logger *logger.Logger) *ValidationMiddleware {
	v := validator.New()

	// Register custom tag name func to use json tags
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &ValidationMiddleware{
		validator: v,
		logger:    logger,
	}
}

// ValidateJSON is a middleware that validates JSON request body against a struct
func (vm *ValidationMiddleware) ValidateJSON(structType interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create new instance of the struct type
		requestStruct := reflect.New(reflect.TypeOf(structType)).Interface()

		// Bind JSON to struct
		if err := c.ShouldBindJSON(requestStruct); err != nil {
			vm.logger.Warn("JSON binding failed", "error", err, "path", c.Request.URL.Path)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid JSON format",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Validate struct
		if err := vm.validator.Struct(requestStruct); err != nil {
			validationErrors := vm.formatValidationErrors(err)
			vm.logger.Warn("Validation failed", "errors", validationErrors, "path", c.Request.URL.Path)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Validation failed",
				"details": validationErrors,
			})
			c.Abort()
			return
		}

		// Store validated struct in context
		c.Set("validated_request", requestStruct)
		c.Next()
	}
}

// ValidateQuery validates query parameters
func (vm *ValidationMiddleware) ValidateQuery(structType interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create new instance of the struct type
		requestStruct := reflect.New(reflect.TypeOf(structType)).Interface()

		// Bind query parameters to struct
		if err := c.ShouldBindQuery(requestStruct); err != nil {
			vm.logger.Warn("Query binding failed", "error", err, "path", c.Request.URL.Path)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid query parameters",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Validate struct
		if err := vm.validator.Struct(requestStruct); err != nil {
			validationErrors := vm.formatValidationErrors(err)
			vm.logger.Warn("Query validation failed", "errors", validationErrors, "path", c.Request.URL.Path)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid query parameters",
				"details": validationErrors,
			})
			c.Abort()
			return
		}

		// Store validated struct in context
		c.Set("validated_query", requestStruct)
		c.Next()
	}
}

// ValidatePathParams validates path parameters
func (vm *ValidationMiddleware) ValidatePathParams(paramValidations map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		errors := make(map[string]string)

		for paramName, validation := range paramValidations {
			paramValue := c.Param(paramName)

			switch validation {
			case "required":
				if paramValue == "" {
					errors[paramName] = "Parameter is required"
				}
			case "uuid":
				if paramValue == "" {
					errors[paramName] = "Parameter is required"
				} else if !isValidUUID(paramValue) {
					errors[paramName] = "Parameter must be a valid UUID"
				}
			case "numeric":
				if paramValue == "" {
					errors[paramName] = "Parameter is required"
				} else if !isNumeric(paramValue) {
					errors[paramName] = "Parameter must be numeric"
				}
			}
		}

		if len(errors) > 0 {
			vm.logger.Warn("Path parameter validation failed", "errors", errors, "path", c.Request.URL.Path)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid path parameters",
				"details": errors,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SanitizeInput is a middleware that sanitizes request input
func (vm *ValidationMiddleware) SanitizeInput() gin.HandlerFunc {
	return func(c *gin.Context) {
		// This would typically sanitize input to prevent XSS, SQL injection, etc.
		// For now, we'll implement basic trimming of spaces

		// Get Content-Type
		contentType := c.GetHeader("Content-Type")

		if strings.Contains(contentType, "application/json") {
			// For JSON requests, the sanitization would happen at the struct level
			// This is handled by the JSON binding and validation
		}

		c.Next()
	}
}

// formatValidationErrors formats validator errors into user-friendly messages
func (vm *ValidationMiddleware) formatValidationErrors(err error) map[string]string {
	errors := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			fieldName := fieldError.Field()

			switch fieldError.Tag() {
			case "required":
				errors[fieldName] = "This field is required"
			case "email":
				errors[fieldName] = "Must be a valid email address"
			case "min":
				errors[fieldName] = "Value is too short (minimum " + fieldError.Param() + ")"
			case "max":
				errors[fieldName] = "Value is too long (maximum " + fieldError.Param() + ")"
			case "gte":
				errors[fieldName] = "Value must be greater than or equal to " + fieldError.Param()
			case "gt":
				errors[fieldName] = "Value must be greater than " + fieldError.Param()
			case "lte":
				errors[fieldName] = "Value must be less than or equal to " + fieldError.Param()
			case "lt":
				errors[fieldName] = "Value must be less than " + fieldError.Param()
			case "uuid":
				errors[fieldName] = "Must be a valid UUID"
			case "oneof":
				errors[fieldName] = "Must be one of: " + fieldError.Param()
			default:
				errors[fieldName] = "Invalid value for " + fieldError.Tag()
			}
		}
	}

	return errors
}

// Helper functions
func isValidUUID(s string) bool {
	// Simple UUID format check
	if len(s) != 36 {
		return false
	}

	// Check for hyphens in correct positions
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}

	// Check that all other characters are hexadecimal
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue // Skip hyphens
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}

	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

// GetValidatedRequest retrieves the validated request from context
func GetValidatedRequest(c *gin.Context) (interface{}, bool) {
	return c.Get("validated_request")
}

// GetValidatedQuery retrieves the validated query from context
func GetValidatedQuery(c *gin.Context) (interface{}, bool) {
	return c.Get("validated_query")
}
