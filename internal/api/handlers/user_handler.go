package handlers

import (
	"easy-orders-backend/internal/api/middleware"
	"net/http"
	"strings"

	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService services.UserService
	logger      *logger.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService services.UserService, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// CreateUser godoc
// @Summary Create a new user
// @Description Register a new user account
// @Tags users
// @Accept json
// @Produce json
// @Param user body services.CreateUserRequest true "User registration details"
// @Success 201 {object} services.UserResponse "User created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request body or validation error"
// @Failure 409 {object} map[string]interface{} "User already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	// Get validated request from context
	validatedReq, exists := middleware.GetValidatedRequest(c)
	if !exists {
		h.logger.Error("Validated request not found in context")
		appErr := errors.NewValidationError("Request validation failed")
		middleware.AbortWithError(c, appErr)
		return
	}

	// Type assert to the expected request type
	req := *validatedReq.(*services.CreateUserRequest)

	user, err := h.userService.CreateUser(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to create user", "error", err, "email", req.Email)

		// Handle specific error types
		if strings.Contains(err.Error(), "already exists") {
			appErr := errors.NewDuplicateError("User", "email", req.Email)
			middleware.AbortWithError(c, appErr)
			return
		}

		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			appErr := errors.NewValidationError(err.Error())
			middleware.AbortWithError(c, appErr)
			return
		}

		appErr := errors.NewInternalError("Failed to create user", err)
		middleware.AbortWithError(c, appErr)
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetUser godoc
// @Summary Get user by ID
// @Description Retrieve user details by user ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} services.UserResponse "User details"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	// Path parameter validation is done by middleware
	id := c.Param("id")

	user, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get user", "error", err, "id", id)

		if strings.Contains(err.Error(), "not found") {
			appErr := errors.NewNotFoundErrorWithID("User", id)
			middleware.AbortWithError(c, appErr)
			return
		}

		appErr := errors.NewInternalError("Failed to get user", err)
		middleware.AbortWithError(c, appErr)
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser godoc
// @Summary Update user
// @Description Update user profile information
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body services.UpdateUserRequest true "Updated user details"
// @Success 200 {object} services.UserResponse "User updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// Path parameter validation is done by middleware
	id := c.Param("id")

	// Get validated request from context
	validatedReq, exists := middleware.GetValidatedRequest(c)
	if !exists {
		h.logger.Error("Validated request not found in context")
		appErr := errors.NewValidationError("Request validation failed")
		middleware.AbortWithError(c, appErr)
		return
	}

	// Type assert to the expected request type
	req := *validatedReq.(*services.UpdateUserRequest)

	user, err := h.userService.UpdateUser(c.Request.Context(), id, req)
	if err != nil {
		h.logger.Error("Failed to update user", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// AuthenticateUser godoc
// @Summary Login user
// @Description Authenticate user and get JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body object{email=string,password=string} true "Login credentials"
// @Success 200 {object} services.AuthResponse "Authentication successful"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 401 {object} map[string]interface{} "Invalid credentials"
// @Router /auth/login [post]
func (h *UserHandler) AuthenticateUser(c *gin.Context) {
	// Define inline struct matching the one in routes
	type LoginRequest struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
	}

	// Get validated request from context
	validatedReq, exists := middleware.GetValidatedRequest(c)
	if !exists {
		h.logger.Error("Validated request not found in context")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request validation failed"})
		return
	}

	// Type assert to the expected request type
	req := *validatedReq.(*LoginRequest)

	auth, err := h.userService.AuthenticateUser(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.Error("Failed to authenticate user", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, auth)
}
