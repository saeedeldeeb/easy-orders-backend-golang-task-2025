package routes

import (
	"easy-orders-backend/internal/api/handlers"
	"easy-orders-backend/internal/api/middleware"
	"easy-orders-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// RegisterOrderRoutes registers all order-related routes
func RegisterOrderRoutes(router *gin.RouterGroup, handler *handlers.OrderHandler, validationMw *middleware.ValidationMiddleware) {
	orders := router.Group("/orders")
	{
		orders.POST("",
			validationMw.ValidateJSON(services.CreateOrderRequest{}),
			handler.CreateOrder,
		)
		orders.GET("",
			validationMw.ValidateQuery(services.ListOrdersRequest{}),
			handler.ListOrders,
		)
		orders.GET("/:id",
			validationMw.ValidatePathParams(map[string]string{"id": "required"}),
			handler.GetOrder,
		)
		orders.GET("/:id/status",
			validationMw.ValidatePathParams(map[string]string{"id": "required"}),
			handler.GetOrderStatus,
		)
		orders.PATCH("/:id/cancel",
			validationMw.ValidatePathParams(map[string]string{"id": "required"}),
			handler.CancelOrder,
		)
	}
}
