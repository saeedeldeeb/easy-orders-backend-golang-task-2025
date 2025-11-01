package routes

import (
	"easy-orders-backend/internal/api/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterPaymentRoutes registers all payment-related routes
func RegisterPaymentRoutes(router *gin.RouterGroup, handler *handlers.PaymentHandler) {
	payments := router.Group("/payments")
	{
		payments.POST("", handler.ProcessPayment)
		payments.GET("/:id", handler.GetPayment)
		payments.POST("/:id/refund", handler.RefundPayment)
	}

	// Order payment routes
	router.GET("/payments/order/:order_id", handler.GetOrderPayments)
}
