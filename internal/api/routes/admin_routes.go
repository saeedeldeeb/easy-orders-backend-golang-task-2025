package routes

import (
	"easy-orders-backend/internal/api/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes registers all admin-related routes
func RegisterAdminRoutes(router *gin.RouterGroup, adminHandler *handlers.AdminHandler, inventoryHandler *handlers.InventoryHandler) {
	admin := router.Group("/admin")
	{
		// Order management
		orders := admin.Group("/orders")
		{
			orders.GET("", adminHandler.GetAllOrders)
			orders.PATCH("/:id/status", adminHandler.UpdateOrderStatus)
		}

		// Reports - Only daily sales report as per README requirement
		reports := admin.Group("/reports")
		{
			reports.GET("/daily", adminHandler.GenerateDailySalesReport)
		}

		// Inventory - Low stock alerts as per README requirement
		inventory := admin.Group("/inventory")
		{
			inventory.GET("/low-stock", inventoryHandler.GetLowStockAlert)
		}
	}
}
