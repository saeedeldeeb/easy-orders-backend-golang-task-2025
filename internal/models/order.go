package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderStatus defines the status of an order
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusFailed    OrderStatus = "failed"
)

// Order represents an order in the system
type Order struct {
	ID          string         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      string         `gorm:"type:uuid;not null;index" json:"user_id" validate:"required"`
	Status      OrderStatus    `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	TotalAmount float64        `gorm:"type:decimal(10,2);not null" json:"total_amount" validate:"gte=0"`
	Currency    string         `gorm:"type:varchar(3);not null;default:'USD'" json:"currency"`
	Notes       string         `gorm:"type:text" json:"notes"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User      *User       `gorm:"foreignKey:UserID;constraint:OnDelete:RESTRICT" json:"user,omitempty"`
	Items     []OrderItem `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	Payments  []Payment   `gorm:"foreignKey:OrderID;constraint:OnDelete:RESTRICT" json:"payments,omitempty"`
	AuditLogs []AuditLog  `gorm:"foreignKey:EntityID;constraint:OnDelete:CASCADE" json:"audit_logs,omitempty"`
}

// BeforeCreate hook to generate UUID if not provided
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	return nil
}

// TableName returns the table name for Order model
func (Order) TableName() string {
	return "orders"
}

// IsPending returns true if order is in pending status
func (o *Order) IsPending() bool {
	return o.Status == OrderStatusPending
}

// IsConfirmed returns true if order is confirmed
func (o *Order) IsConfirmed() bool {
	return o.Status == OrderStatusConfirmed
}

// IsPaid returns true if order is paid
func (o *Order) IsPaid() bool {
	return o.Status == OrderStatusPaid
}

// IsCancellable returns true if order can be cancelled
func (o *Order) IsCancellable() bool {
	return o.Status == OrderStatusPending || o.Status == OrderStatusConfirmed
}

// IsCompletable returns true if order can be marked as completed
func (o *Order) IsCompletable() bool {
	return o.Status == OrderStatusShipped
}

// CanTransitionTo checks if order can transition to the given status
func (o *Order) CanTransitionTo(newStatus OrderStatus) bool {
	switch o.Status {
	case OrderStatusPending:
		return newStatus == OrderStatusConfirmed || newStatus == OrderStatusCancelled || newStatus == OrderStatusFailed
	case OrderStatusConfirmed:
		return newStatus == OrderStatusPaid || newStatus == OrderStatusCancelled
	case OrderStatusPaid:
		return newStatus == OrderStatusShipped || newStatus == OrderStatusCancelled
	case OrderStatusShipped:
		return newStatus == OrderStatusDelivered
	case OrderStatusDelivered, OrderStatusCancelled, OrderStatusFailed:
		return false // Terminal states
	default:
		return false
	}
}

// CalculateTotal calculates the total amount from order items
func (o *Order) CalculateTotal() float64 {
	var total float64
	for _, item := range o.Items {
		total += item.UnitPrice * float64(item.Quantity)
	}
	return total
}
