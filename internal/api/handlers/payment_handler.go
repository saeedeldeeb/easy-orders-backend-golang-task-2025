package handlers

import (
	"net/http"
	"strings"

	"easy-orders-backend/internal/api/middleware"
	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// PaymentHandler handles payment-related HTTP requests
type PaymentHandler struct {
	paymentService services.PaymentService
	logger         *logger.Logger
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(paymentService services.PaymentService, logger *logger.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		logger:         logger,
	}
}

// ProcessPayment godoc
// @Summary Process a payment
// @Description Process payment for an order
// @Tags payments
// @Accept json
// @Produce json
// @Param payment body services.ProcessPaymentRequest true "Payment details"
// @Success 201 {object} object{message=string,data=services.PaymentResponse} "Payment processed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 409 {object} map[string]interface{} "Order already paid"
// @Failure 402 {object} map[string]interface{} "Payment processing failed"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /payments [post]
func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	h.logger.Debug("Processing payment via API")

	// Get validated request from context
	validatedReq, exists := middleware.GetValidatedRequest(c)
	if !exists {
		h.logger.Error("Validated request not found in context")
		appErr := errors.NewValidationError("Request validation failed")
		middleware.AbortWithError(c, appErr)
		return
	}

	// Type assert to the expected request type
	req := *validatedReq.(*services.ProcessPaymentRequest)

	// Call service
	payment, err := h.paymentService.ProcessPayment(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to process payment", "error", err, "order_id", req.OrderID)

		// Handle specific error types
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found",
			})
			return
		}

		if strings.Contains(err.Error(), "already been paid") {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Order has already been paid",
			})
			return
		}

		if strings.Contains(err.Error(), "does not match") || strings.Contains(err.Error(), "cannot be paid") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		if strings.Contains(err.Error(), "processing failed") {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": "Payment processing failed",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process payment",
		})
		return
	}

	h.logger.Info("Payment processed successfully via API", "id", payment.ID, "order_id", req.OrderID, "amount", req.Amount)
	c.JSON(http.StatusCreated, gin.H{
		"message": "Payment processed successfully",
		"data":    payment,
	})
}

// GetPayment godoc
// @Summary Get payment by ID
// @Description Retrieve payment details by payment ID
// @Tags payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} object{data=services.PaymentResponse} "Payment details"
// @Failure 400 {object} map[string]interface{} "Invalid payment ID"
// @Failure 404 {object} map[string]interface{} "Payment not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /payments/{id} [get]
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	// Path parameter validation is done by middleware
	paymentID := c.Param("id")
	h.logger.Debug("Getting payment via API", "id", paymentID)

	// Call service
	payment, err := h.paymentService.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		h.logger.Error("Failed to get payment", "error", err, "id", paymentID)

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Payment not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get payment",
		})
		return
	}

	h.logger.Debug("Payment retrieved successfully via API", "id", paymentID)
	c.JSON(http.StatusOK, gin.H{
		"data": payment,
	})
}

// RefundPayment godoc
// @Summary Refund a payment
// @Description Process a refund for a payment
// @Tags payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID"
// @Param refund body object{amount=number} true "Refund amount"
// @Success 201 {object} object{message=string,data=services.PaymentResponse} "Refund processed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Payment not found"
// @Failure 409 {object} map[string]interface{} "Payment cannot be refunded"
// @Failure 402 {object} map[string]interface{} "Refund processing failed"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /payments/{id}/refund [post]
func (h *PaymentHandler) RefundPayment(c *gin.Context) {
	// Path parameter validation is done by middleware
	paymentID := c.Param("id")
	h.logger.Debug("Processing refund via API", "payment_id", paymentID)

	// Define inline struct matching the one in routes
	type RefundRequest struct {
		Amount float64 `json:"amount" validate:"required,gt=0"`
	}

	// Get validated request from context
	validatedReq, exists := middleware.GetValidatedRequest(c)
	if !exists {
		h.logger.Error("Validated request not found in context")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request validation failed"})
		return
	}

	// Type assert to the expected request type
	req := *validatedReq.(*RefundRequest)

	// Call service
	refund, err := h.paymentService.RefundPayment(c.Request.Context(), paymentID, req.Amount)
	if err != nil {
		h.logger.Error("Failed to process refund", "error", err, "payment_id", paymentID, "amount", req.Amount)

		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Payment not found",
			})
			return
		}

		if strings.Contains(err.Error(), "cannot be refunded") {
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
			return
		}

		if strings.Contains(err.Error(), "cannot exceed") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		if strings.Contains(err.Error(), "processing failed") {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": "Refund processing failed",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process refund",
		})
		return
	}

	h.logger.Info("Refund processed successfully via API", "refund_id", refund.ID, "payment_id", paymentID, "amount", req.Amount)
	c.JSON(http.StatusCreated, gin.H{
		"message": "Refund processed successfully",
		"data":    refund,
	})
}

// GetOrderPayments godoc
// @Summary Get order payments
// @Description Get all payments for a specific order
// @Tags payments
// @Accept json
// @Produce json
// @Param order_id path string true "Order ID"
// @Success 200 {object} object{data=[]services.PaymentResponse} "Order payments"
// @Failure 400 {object} map[string]interface{} "Invalid order ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security BearerAuth
// @Router /orders/{order_id}/payments [get]
func (h *PaymentHandler) GetOrderPayments(c *gin.Context) {
	// Path parameter validation is done by middleware
	orderID := c.Param("order_id")
	h.logger.Debug("Getting order payments via API", "order_id", orderID)

	// Call service
	payments, err := h.paymentService.GetOrderPayments(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get order payments", "error", err, "order_id", orderID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get order payments",
		})
		return
	}

	h.logger.Debug("Order payments retrieved successfully via API", "order_id", orderID, "count", len(payments))
	c.JSON(http.StatusOK, gin.H{
		"data": payments,
	})
}
