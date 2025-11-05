package routes

import (
	"easy-orders-backend/internal/api/handlers"
	"easy-orders-backend/internal/api/middleware"
	"easy-orders-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// RegisterProductRoutes registers all product-related routes
func RegisterProductRoutes(router *gin.RouterGroup, productHandler *handlers.ProductHandler, inventoryHandler *handlers.InventoryHandler, validationMw *middleware.ValidationMiddleware) {
	products := router.Group("/products")
	{
		products.POST("",
			validationMw.ValidateJSON(services.CreateProductRequest{}),
			productHandler.CreateProduct,
		)
		products.GET("",
			validationMw.ValidateQuery(services.ListProductsRequest{}),
			productHandler.ListProducts,
		)
		products.GET("/:id",
			validationMw.ValidatePathParams(map[string]string{"id": "required"}),
			productHandler.GetProduct,
		)
		products.PUT("/:id",
			validationMw.ValidatePathParams(map[string]string{"id": "required"}),
			validationMw.ValidateJSON(services.UpdateProductRequest{}),
			productHandler.UpdateProduct,
		)

		// Inventory check endpoint (as per README requirement)
		products.GET("/:id/inventory",
			validationMw.ValidatePathParams(map[string]string{"id": "required"}),
			inventoryHandler.CheckAvailability,
		)
	}
}
