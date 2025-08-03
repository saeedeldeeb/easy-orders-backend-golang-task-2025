package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// InventoryHandler handles inventory-related HTTP requests
type InventoryHandler struct {
	inventoryService services.InventoryService
	logger           *logger.Logger
}

// NewInventoryHandler creates a new inventory handler
func NewInventoryHandler(inventoryService services.InventoryService, logger *logger.Logger) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
		logger:           logger,
	}
}

// CheckAvailability handles GET /api/v1/inventory/check/:product_id
func (h *InventoryHandler) CheckAvailability(c *gin.Context) {
	productID := c.Param("product_id")
	h.logger.Debug("Checking inventory availability via API", "product_id", productID)

	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Product ID is required",
		})
		return
	}

	quantityStr := c.Query("quantity")
	if quantityStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Quantity parameter is required",
		})
		return
	}

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid quantity parameter",
		})
		return
	}

	// Call service
	available, err := h.inventoryService.CheckAvailability(c.Request.Context(), productID, quantity)
	if err != nil {
		h.logger.Error("Failed to check inventory availability", "error", err, "product_id", productID, "quantity", quantity)

		if strings.Contains(err.Error(), "required") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check inventory availability",
		})
		return
	}

	h.logger.Debug("Inventory availability checked via API", "product_id", productID, "quantity", quantity, "available", available)
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"product_id": productID,
			"quantity":   quantity,
			"available":  available,
		},
	})
}

// ReserveInventory handles POST /api/v1/inventory/reserve
func (h *InventoryHandler) ReserveInventory(c *gin.Context) {
	h.logger.Debug("Reserving inventory via API")

	var req struct {
		Items []services.InventoryItem `json:"items" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind inventory reservation request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Call service
	err := h.inventoryService.ReserveInventory(c.Request.Context(), req.Items)
	if err != nil {
		h.logger.Error("Failed to reserve inventory", "error", err, "items_count", len(req.Items))

		if strings.Contains(err.Error(), "insufficient stock") {
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
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
			"error": "Failed to reserve inventory",
		})
		return
	}

	h.logger.Info("Inventory reserved successfully via API", "items_count", len(req.Items))
	c.JSON(http.StatusOK, gin.H{
		"message": "Inventory reserved successfully",
	})
}

// ReleaseInventory handles POST /api/v1/inventory/release
func (h *InventoryHandler) ReleaseInventory(c *gin.Context) {
	h.logger.Debug("Releasing inventory via API")

	var req struct {
		Items []services.InventoryItem `json:"items" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind inventory release request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Call service
	err := h.inventoryService.ReleaseInventory(c.Request.Context(), req.Items)
	if err != nil {
		h.logger.Error("Failed to release inventory", "error", err, "items_count", len(req.Items))

		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to release inventory",
		})
		return
	}

	h.logger.Info("Inventory released successfully via API", "items_count", len(req.Items))
	c.JSON(http.StatusOK, gin.H{
		"message": "Inventory released successfully",
	})
}

// UpdateStock handles PUT /api/v1/inventory/:product_id
func (h *InventoryHandler) UpdateStock(c *gin.Context) {
	productID := c.Param("product_id")
	h.logger.Debug("Updating stock via API", "product_id", productID)

	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Product ID is required",
		})
		return
	}

	var req struct {
		Quantity int `json:"quantity" binding:"required,gte=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind stock update request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Call service
	err := h.inventoryService.UpdateStock(c.Request.Context(), productID, req.Quantity)
	if err != nil {
		h.logger.Error("Failed to update stock", "error", err, "product_id", productID, "quantity", req.Quantity)

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}

		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "cannot be negative") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update stock",
		})
		return
	}

	h.logger.Info("Stock updated successfully via API", "product_id", productID, "new_quantity", req.Quantity)
	c.JSON(http.StatusOK, gin.H{
		"message": "Stock updated successfully",
	})
}

// GetLowStockAlert handles GET /api/v1/inventory/low-stock
func (h *InventoryHandler) GetLowStockAlert(c *gin.Context) {
	h.logger.Debug("Getting low stock alert via API")

	// Parse threshold parameter
	threshold := 10 // Default threshold
	if thresholdStr := c.Query("threshold"); thresholdStr != "" {
		if t, err := strconv.Atoi(thresholdStr); err == nil && t >= 0 {
			threshold = t
		}
	}

	// Call service
	response, err := h.inventoryService.GetLowStockAlert(c.Request.Context(), threshold)
	if err != nil {
		h.logger.Error("Failed to get low stock alert", "error", err, "threshold", threshold)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get low stock alert",
		})
		return
	}

	h.logger.Debug("Low stock alert retrieved successfully via API", "threshold", threshold, "alert_count", response.Count)
	c.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

// RegisterRoutes registers all inventory routes
func (h *InventoryHandler) RegisterRoutes(router *gin.RouterGroup) {
	inventory := router.Group("/inventory")
	{
		inventory.GET("/check/:product_id", h.CheckAvailability)
		inventory.POST("/reserve", h.ReserveInventory)
		inventory.POST("/release", h.ReleaseInventory)
		inventory.PUT("/:product_id", h.UpdateStock)
		inventory.GET("/low-stock", h.GetLowStockAlert)
	}
}
