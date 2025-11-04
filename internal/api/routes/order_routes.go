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
		orders.GET("/:id/status", handler.GetOrderStatus)
		orders.PATCH("/:id/cancel", handler.CancelOrder)
	}
}
