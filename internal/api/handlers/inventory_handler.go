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

// CheckAvailability godoc
// @Summary Check inventory availability
// @Description Check if a specific quantity of a product is available
// @Tags inventory
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param quantity query int true "Quantity to check"
// @Success 200 {object} map[string]interface{} "Availability status"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /products/{id}/inventory [get]
func (h *InventoryHandler) CheckAvailability(c *gin.Context) {
	productID := c.Param("id")
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

// ReserveInventory godoc
// @Summary Reserve inventory
// @Description Reserve inventory items for an order
// @Tags inventory
// @Accept json
// @Produce json
// @Param items body object{items=[]services.InventoryItem} true "Items to reserve"
// @Success 200 {object} map[string]interface{} "Inventory reserved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 409 {object} map[string]interface{} "Insufficient stock"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /inventory/reserve [post]
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

// ReleaseInventory godoc
// @Summary Release inventory
// @Description Release previously reserved inventory items
// @Tags inventory
// @Accept json
// @Produce json
// @Param items body object{items=[]services.InventoryItem} true "Items to release"
// @Success 200 {object} map[string]interface{} "Inventory released successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /inventory/release [post]
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

// UpdateStock godoc
// @Summary Update stock (Admin)
// @Description Update product stock quantity (Admin only)
// @Tags inventory
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID"
// @Param stock body object{quantity=int} true "New stock quantity"
// @Success 200 {object} map[string]interface{} "Stock updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /inventory/{product_id} [put]
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

// GetLowStockAlert godoc
// @Summary Get low stock alerts (Admin)
// @Description Get products with low stock levels (Admin only)
// @Tags inventory
// @Accept json
// @Produce json
// @Param threshold query int false "Stock threshold" default(10)
// @Success 200 {object} map[string]interface{} "Low stock products"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /inventory/low-stock [get]
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
