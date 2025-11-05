package routes

import (
	"easy-orders-backend/internal/api/handlers"
	"easy-orders-backend/internal/api/middleware"
	"easy-orders-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers all user-related routes
func RegisterUserRoutes(router *gin.RouterGroup, handler *handlers.UserHandler, validationMw *middleware.ValidationMiddleware) {
	// User management routes
	users := router.Group("/users")
	{
		users.POST("",
			validationMw.ValidateJSON(services.CreateUserRequest{}),
			handler.CreateUser,
		)
		users.GET("/:id",
			validationMw.ValidatePathParams(map[string]string{"id": "required"}),
			handler.GetUser,
		)
		users.PUT("/:id",
			validationMw.ValidatePathParams(map[string]string{"id": "required"}),
			validationMw.ValidateJSON(services.UpdateUserRequest{}),
			handler.UpdateUser,
		)
	}

	// Authentication routes
	auth := router.Group("/auth")
	{
		// Define inline struct for login request validation
		type LoginRequest struct {
			Email    string `json:"email" validate:"required,email"`
			Password string `json:"password" validate:"required,min=8"`
		}
		auth.POST("/login",
			validationMw.ValidateJSON(LoginRequest{}),
			handler.AuthenticateUser,
		)
	}
}
