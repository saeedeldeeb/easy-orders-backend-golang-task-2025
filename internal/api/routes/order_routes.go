package routes

import (
	"easy-orders-backend/internal/api/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterOrderRoutes registers all order-related routes
func RegisterOrderRoutes(router *gin.RouterGroup, handler *handlers.OrderHandler) {
	orders := router.Group("/orders")
	{
		orders.POST("", handler.CreateOrder)
		orders.GET("", handler.ListOrders)
		orders.GET("/:id", handler.GetOrder)
		orders.PATCH("/:id/status", handler.UpdateOrderStatus)
		orders.PATCH("/:id/cancel", handler.CancelOrder)
	}

	// User-specific order routes
	router.GET("/orders/user/:user_id", handler.GetUserOrders)
}
