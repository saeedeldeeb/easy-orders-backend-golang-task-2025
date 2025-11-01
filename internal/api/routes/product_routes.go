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
		products.GET("/search", productHandler.SearchProducts)
		products.GET("/:id", productHandler.GetProduct)
		products.PUT("/:id", productHandler.UpdateProduct)
		products.DELETE("/:id", productHandler.DeleteProduct)

		// Inventory check endpoint (as per README requirement)
		// Note: Handler expects "product_id" param, will need to update handler to use "id"
		products.GET("/:product_id/inventory", inventoryHandler.CheckAvailability)
	}
}
