package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ProductHandler handles product-related HTTP requests
type ProductHandler struct {
	productService services.ProductService
	logger         *logger.Logger
}

// NewProductHandler creates a new product handler
func NewProductHandler(productService services.ProductService, logger *logger.Logger) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		logger:         logger,
	}
}

// CreateProduct handles POST /api/v1/products
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	h.logger.Debug("Creating product via API")

	var req services.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind product creation request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Call service
	product, err := h.productService.CreateProduct(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to create product", "error", err, "name", req.Name)

		// Handle specific error types
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Product with this SKU already exists",
			})
			return
		}

		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create product",
		})
		return
	}

	h.logger.Info("Product created successfully via API", "id", product.ID, "name", product.Name)
	c.JSON(http.StatusCreated, gin.H{
		"message": "Product created successfully",
		"data":    product,
	})
}

// GetProduct handles GET /api/v1/products/:id
func (h *ProductHandler) GetProduct(c *gin.Context) {
	productID := c.Param("id")
	h.logger.Debug("Getting product via API", "id", productID)

	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Product ID is required",
		})
		return
	}

	// Call service
	product, err := h.productService.GetProduct(c.Request.Context(), productID)
	if err != nil {
		h.logger.Error("Failed to get product", "error", err, "id", productID)

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get product",
		})
		return
	}

	h.logger.Debug("Product retrieved successfully via API", "id", productID)
	c.JSON(http.StatusOK, gin.H{
		"data": product,
	})
}

// UpdateProduct handles PUT /api/v1/products/:id
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	productID := c.Param("id")
	h.logger.Debug("Updating product via API", "id", productID)

	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Product ID is required",
		})
		return
	}

	var req services.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind product update request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Call service
	product, err := h.productService.UpdateProduct(c.Request.Context(), productID, req)
	if err != nil {
		h.logger.Error("Failed to update product", "error", err, "id", productID)

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update product",
		})
		return
	}

	h.logger.Info("Product updated successfully via API", "id", productID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Product updated successfully",
		"data":    product,
	})
}

// DeleteProduct handles DELETE /api/v1/products/:id
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	productID := c.Param("id")
	h.logger.Debug("Deleting product via API", "id", productID)

	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Product ID is required",
		})
		return
	}

	// Call service
	err := h.productService.DeleteProduct(c.Request.Context(), productID)
	if err != nil {
		h.logger.Error("Failed to delete product", "error", err, "id", productID)

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete product",
		})
		return
	}

	h.logger.Info("Product deleted successfully via API", "id", productID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Product deleted successfully",
	})
}

// ListProducts handles GET /api/v1/products
func (h *ProductHandler) ListProducts(c *gin.Context) {
	h.logger.Debug("Listing products via API")

	// Parse query parameters
	var req services.ListProductsRequest

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			req.Offset = offset
		}
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	// Parse category filter
	if categoryID := c.Query("category_id"); categoryID != "" {
		req.CategoryID = categoryID
	}

	// Parse active only filter
	if activeOnlyStr := c.Query("active_only"); activeOnlyStr != "" {
		req.ActiveOnly = activeOnlyStr == "true"
	}

	// Call service
	response, err := h.productService.ListProducts(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to list products", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list products",
		})
		return
	}

	h.logger.Debug("Products listed successfully via API", "count", len(response.Products))
	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// SearchProducts handles GET /api/v1/products/search
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	h.logger.Debug("Searching products via API")

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Search query is required",
		})
		return
	}

	// Parse query parameters
	var req services.SearchProductsRequest
	req.Query = query

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			req.Offset = offset
		}
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	// Call service
	response, err := h.productService.SearchProducts(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to search products", "error", err, "query", query)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search products",
		})
		return
	}

	h.logger.Debug("Product search completed successfully via API", "query", query, "count", len(response.Products))
	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}
