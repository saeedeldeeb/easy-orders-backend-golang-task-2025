package routes

import (
	"easy-orders-backend/internal/api/handlers"
	"easy-orders-backend/internal/api/middleware"
	"easy-orders-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes registers all admin-related routes
func RegisterAdminRoutes(router *gin.RouterGroup, adminHandler *handlers.AdminHandler, inventoryHandler *handlers.InventoryHandler, validationMw *middleware.ValidationMiddleware) {
	admin := router.Group("/admin")
	{
		// Order management
		orders := admin.Group("/orders")
		{
			orders.GET("",
				validationMw.ValidateQuery(services.ListOrdersRequest{}),
				adminHandler.GetAllOrders,
			)

			orders.PATCH("/:id/status",
				validationMw.ValidatePathParams(map[string]string{"id": "required"}),
				validationMw.ValidateJSON(services.UpdateStatusRequest{}),
				adminHandler.UpdateOrderStatus,
			)
		}

		// Reports - Only daily sales report as per README requirement
		reports := admin.Group("/reports")
		{
			reports.GET("/daily",
				validationMw.ValidateQuery(services.DailySalesReportQuery{}),
				adminHandler.GenerateDailySalesReport,
			)
		}

		// Inventory - Low stock alerts as per README requirement
		inventory := admin.Group("/inventory")
		{
			inventory.GET("/low-stock",
				validationMw.ValidateQuery(services.LowStockQuery{}),
				inventoryHandler.GetLowStockAlert,
			)
		}
	}
}
