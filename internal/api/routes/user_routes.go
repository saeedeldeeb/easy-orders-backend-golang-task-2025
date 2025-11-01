package routes

import (
	"easy-orders-backend/internal/api/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers all user-related routes
func RegisterUserRoutes(router *gin.RouterGroup, handler *handlers.UserHandler) {
	// User management routes
	users := router.Group("/users")
	{
		users.POST("", handler.CreateUser)
		users.GET("", handler.ListUsers)
		users.GET("/:id", handler.GetUser)
		users.PUT("/:id", handler.UpdateUser)
		users.DELETE("/:id", handler.DeleteUser)
	}

	// Authentication routes
	auth := router.Group("/auth")
	{
		auth.POST("/login", handler.AuthenticateUser)
	}
}
