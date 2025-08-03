package repository

import (
	"context"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/logger"

	"gorm.io/gorm"
)

// paymentRepository implements PaymentRepository interface
type paymentRepository struct {
	db     *database.DB
	logger *logger.Logger
}

// NewPaymentRepository creates a new payment repository
func NewPaymentRepository(db *database.DB, logger *logger.Logger) PaymentRepository {
	return &paymentRepository{
		db:     db,
		logger: logger,
	}
}

func (r *paymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	r.logger.Debug("Creating payment in database", "order_id", payment.OrderID, "amount", payment.Amount, "method", payment.Method)

	if err := r.db.WithContext(ctx).Create(payment).Error; err != nil {
		r.logger.Error("Failed to create payment", "error", err, "order_id", payment.OrderID)
		return err
	}

	r.logger.Info("Payment created in database", "id", payment.ID, "transaction_id", payment.TransactionID, "order_id", payment.OrderID)
	return nil
}

func (r *paymentRepository) GetByID(ctx context.Context, id string) (*models.Payment, error) {
	r.logger.Debug("Getting payment by ID", "id", id)

	var payment models.Payment
	if err := r.db.WithContext(ctx).
		Preload("Order").
		Preload("Order.User").
		First(&payment, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Debug("Payment not found", "id", id)
			return nil, nil
		}
		r.logger.Error("Failed to get payment by ID", "error", err, "id", id)
		return nil, err
	}

	r.logger.Debug("Payment retrieved from database", "id", id, "status", payment.Status)
	return &payment, nil
}

func (r *paymentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error) {
	r.logger.Debug("Getting payment by transaction ID", "transaction_id", transactionID)

	var payment models.Payment
	if err := r.db.WithContext(ctx).
		Preload("Order").
		Preload("Order.User").
		First(&payment, "transaction_id = ?", transactionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Debug("Payment not found", "transaction_id", transactionID)
			return nil, nil
		}
		r.logger.Error("Failed to get payment by transaction ID", "error", err, "transaction_id", transactionID)
		return nil, err
	}

	r.logger.Debug("Payment retrieved from database", "transaction_id", transactionID, "status", payment.Status)
	return &payment, nil
}

func (r *paymentRepository) GetByOrderID(ctx context.Context, orderID string) ([]*models.Payment, error) {
	r.logger.Debug("Getting payments by order ID", "order_id", orderID)

	var payments []*models.Payment
	if err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at DESC").
		Find(&payments).Error; err != nil {
		r.logger.Error("Failed to get payments by order ID", "error", err, "order_id", orderID)
		return nil, err
	}

	r.logger.Debug("Payments retrieved for order", "order_id", orderID, "count", len(payments))
	return payments, nil
}

func (r *paymentRepository) Update(ctx context.Context, payment *models.Payment) error {
	r.logger.Debug("Updating payment in database", "id", payment.ID)

	if err := r.db.WithContext(ctx).Save(payment).Error; err != nil {
		r.logger.Error("Failed to update payment", "error", err, "id", payment.ID)
		return err
	}

	r.logger.Info("Payment updated in database", "id", payment.ID, "status", payment.Status)
	return nil
}

func (r *paymentRepository) UpdateStatus(ctx context.Context, id string, status models.PaymentStatus) error {
	r.logger.Debug("Updating payment status", "id", id, "status", status)

	result := r.db.WithContext(ctx).
		Model(&models.Payment{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		r.logger.Error("Failed to update payment status", "error", result.Error, "id", id)
		return result.Error
	}

	if result.RowsAffected == 0 {
		r.logger.Warn("No payment found to update status", "id", id)
		return gorm.ErrRecordNotFound
	}

	r.logger.Info("Payment status updated", "id", id, "status", status)
	return nil
}

func (r *paymentRepository) List(ctx context.Context, offset, limit int) ([]*models.Payment, error) {
	r.logger.Debug("Listing payments from database", "offset", offset, "limit", limit)

	var payments []*models.Payment
	if err := r.db.WithContext(ctx).
		Preload("Order").
		Preload("Order.User").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&payments).Error; err != nil {
		r.logger.Error("Failed to list payments", "error", err)
		return nil, err
	}

	r.logger.Debug("Payments retrieved from database", "count", len(payments))
	return payments, nil
}
