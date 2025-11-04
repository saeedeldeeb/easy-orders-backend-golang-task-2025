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
// @Tags products
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

// GetLowStockAlert godoc
// @Summary Get low stock alerts (Admin)
// @Description Get products with low stock levels (Admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Param threshold query int false "Stock threshold" default(10)
// @Success 200 {object} map[string]interface{} "Low stock products"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /admin/inventory/low-stock [get]
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
