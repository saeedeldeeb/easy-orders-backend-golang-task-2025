package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentStatus defines the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusProcessed PaymentStatus = "processed"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// PaymentMethod defines the payment method
type PaymentMethod string

const (
	PaymentMethodCreditCard   PaymentMethod = "credit_card"
	PaymentMethodDebitCard    PaymentMethod = "debit_card"
	PaymentMethodPayPal       PaymentMethod = "paypal"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCash         PaymentMethod = "cash"
)

// Payment represents a payment transaction
type Payment struct {
	ID                string        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrderID           string        `gorm:"type:uuid;not null;index" json:"order_id" validate:"required"`
	Amount            float64       `gorm:"type:decimal(10,2);not null" json:"amount" validate:"required,gt=0"`
	Currency          string        `gorm:"type:varchar(3);not null;default:'USD'" json:"currency"`
	Status            PaymentStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	Method            PaymentMethod `gorm:"type:varchar(20);not null" json:"method" validate:"required"`
	TransactionID     string        `gorm:"type:varchar(255);uniqueIndex" json:"transaction_id"`
	ExternalReference string        `gorm:"type:varchar(255)" json:"external_reference"`
	FailureReason     string        `gorm:"type:text" json:"failure_reason"`
	ProcessedAt       *time.Time    `json:"processed_at"`

	// Idempotency and retry fields
	IdempotencyKey string     `gorm:"type:varchar(255);uniqueIndex" json:"idempotency_key"`
	AttemptCount   int        `gorm:"default:0" json:"attempt_count"`
	MaxRetries     int        `gorm:"default:3" json:"max_retries"`
	NextRetryAt    *time.Time `json:"next_retry_at"`
	LastAttemptAt  *time.Time `json:"last_attempt_at"`

	// Gateway information
	Gateway       string  `gorm:"type:varchar(50)" json:"gateway"`
	GatewayTxnID  string  `gorm:"type:varchar(255)" json:"gateway_txn_id"`
	ProcessingFee float64 `gorm:"type:decimal(10,2);default:0" json:"processing_fee"`

	// Metadata and audit
	Metadata string `gorm:"type:jsonb" json:"metadata"` // JSON field for additional data

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Order     *Order     `gorm:"foreignKey:OrderID;constraint:OnDelete:RESTRICT" json:"order,omitempty"`
	AuditLogs []AuditLog `gorm:"foreignKey:EntityID;constraint:OnDelete:CASCADE" json:"audit_logs,omitempty"`
}

// BeforeCreate hook to generate UUID and transaction ID
func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	if p.TransactionID == "" {
		p.TransactionID = "TXN_" + uuid.New().String()
	}
	return nil
}

// TableName returns the table name for Payment model
func (Payment) TableName() string {
	return "payments"
}

// IsPending returns true if payment is pending
func (p *Payment) IsPending() bool {
	return p.Status == PaymentStatusPending
}

// IsCompleted returns true if payment is completed
func (p *Payment) IsCompleted() bool {
	return p.Status == PaymentStatusCompleted
}

// IsFailed returns true if payment failed
func (p *Payment) IsFailed() bool {
	return p.Status == PaymentStatusFailed
}

// IsRefunded returns true if payment is refunded
func (p *Payment) IsRefunded() bool {
	return p.Status == PaymentStatusRefunded
}

// CanRefund returns true if payment can be refunded
func (p *Payment) CanRefund() bool {
	return p.Status == PaymentStatusCompleted
}

// CanRetry returns true if payment can be retried
func (p *Payment) CanRetry() bool {
	return p.Status == PaymentStatusFailed || p.Status == PaymentStatusCancelled
}

// MarkProcessed marks the payment as processed
func (p *Payment) MarkProcessed() {
	p.Status = PaymentStatusProcessed
	now := time.Now()
	p.ProcessedAt = &now
}

// MarkCompleted marks the payment as completed
func (p *Payment) MarkCompleted() {
	p.Status = PaymentStatusCompleted
	if p.ProcessedAt == nil {
		now := time.Now()
		p.ProcessedAt = &now
	}
}

// MarkFailed marks the payment as failed with a reason
func (p *Payment) MarkFailed(reason string) {
	p.Status = PaymentStatusFailed
	p.FailureReason = reason
	now := time.Now()
	p.LastAttemptAt = &now
}

// CanRetryAt checks if payment can be retried at the specified time
func (p *Payment) CanRetryAt(t time.Time) bool {
	if !p.CanRetry() {
		return false
	}
	if p.AttemptCount >= p.MaxRetries {
		return false
	}
	if p.NextRetryAt != nil && t.Before(*p.NextRetryAt) {
		return false
	}
	return true
}

// IncrementAttempt increments the attempt count and sets the last attempt time
func (p *Payment) IncrementAttempt() {
	p.AttemptCount++
	now := time.Now()
	p.LastAttemptAt = &now
}

// SetNextRetryAt sets when the next retry attempt should occur
func (p *Payment) SetNextRetryAt(t time.Time) {
	p.NextRetryAt = &t
}

// IsRetryDue checks if a retry is due now
func (p *Payment) IsRetryDue() bool {
	if !p.CanRetry() {
		return false
	}
	if p.AttemptCount >= p.MaxRetries {
		return false
	}
	if p.NextRetryAt == nil {
		return true
	}
	return time.Now().After(*p.NextRetryAt)
}

// HasIdempotencyKey checks if payment has an idempotency key
func (p *Payment) HasIdempotencyKey() bool {
	return p.IdempotencyKey != ""
}
