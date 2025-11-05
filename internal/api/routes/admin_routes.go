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

			// Define inline struct for status update request validation
			type UpdateStatusRequest struct {
				Status string `json:"status" validate:"required"`
			}
			orders.PATCH("/:id/status",
				validationMw.ValidatePathParams(map[string]string{"id": "required"}),
				validationMw.ValidateJSON(UpdateStatusRequest{}),
				adminHandler.UpdateOrderStatus,
			)
		}

		// Reports - Only daily sales report as per README requirement
		reports := admin.Group("/reports")
		{
			// Define inline struct for date query parameter validation
			type DailySalesReportQuery struct {
				Date string `form:"date"`
			}
			reports.GET("/daily",
				validationMw.ValidateQuery(DailySalesReportQuery{}),
				adminHandler.GenerateDailySalesReport,
			)
		}

		// Inventory - Low stock alerts as per README requirement
		inventory := admin.Group("/inventory")
		{
			// Define inline struct for threshold query parameter validation
			type LowStockQuery struct {
				Threshold int `form:"threshold" validate:"omitempty,gte=0"`
			}
			inventory.GET("/low-stock",
				validationMw.ValidateQuery(LowStockQuery{}),
				inventoryHandler.GetLowStockAlert,
			)
		}
	}
}
