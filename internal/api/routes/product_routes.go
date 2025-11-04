package routes

import (
	"easy-orders-backend/internal/api/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterProductRoutes registers all product-related routes
func RegisterProductRoutes(router *gin.RouterGroup, productHandler *handlers.ProductHandler, inventoryHandler *handlers.InventoryHandler) {
	products := router.Group("/products")
	{
		products.POST("", productHandler.CreateProduct)
		products.GET("", productHandler.ListProducts)
		products.GET("/:id", productHandler.GetProduct)
		products.PUT("/:id", productHandler.UpdateProduct)

		// Inventory check endpoint (as per README requirement)
		products.GET("/:id/inventory", inventoryHandler.CheckAvailability)
	}
}
