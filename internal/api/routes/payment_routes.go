package routes

import (
	"easy-orders-backend/internal/api/handlers"
	"easy-orders-backend/internal/api/middleware"
	"easy-orders-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// RegisterPaymentRoutes registers all payment-related routes
func RegisterPaymentRoutes(router *gin.RouterGroup, handler *handlers.PaymentHandler, validationMw *middleware.ValidationMiddleware) {
	payments := router.Group("/payments")
	{
		payments.POST("",
			validationMw.ValidateJSON(services.ProcessPaymentRequest{}),
			handler.ProcessPayment,
		)
		payments.GET("/:id",
			validationMw.ValidatePathParams(map[string]string{"id": "required"}),
			handler.GetPayment,
		)
	}

	// Order payment routes
	router.GET("/payments/order/:order_id",
		validationMw.ValidatePathParams(map[string]string{"order_id": "required"}),
		handler.GetOrderPayments,
	)
}
