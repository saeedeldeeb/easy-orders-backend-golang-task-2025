package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderItem represents individual items within an order
type OrderItem struct {
	ID         string    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrderID    string    `gorm:"type:uuid;not null;index" json:"order_id" validate:"required"`
	ProductID  string    `gorm:"type:uuid;not null;index" json:"product_id" validate:"required"`
	Quantity   int       `gorm:"not null" json:"quantity" validate:"required,gt=0"`
	UnitPrice  float64   `gorm:"type:decimal(10,2);not null" json:"unit_price" validate:"required,gt=0"`
	TotalPrice float64   `gorm:"type:decimal(10,2);not null" json:"total_price" validate:"gte=0"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Relationships
	Order   *Order   `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"order,omitempty"`
	Product *Product `gorm:"foreignKey:ProductID;constraint:OnDelete:RESTRICT" json:"product,omitempty"`
}

// BeforeCreate hook to generate UUID and calculate total price
func (oi *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == "" {
		oi.ID = uuid.New().String()
	}
	oi.TotalPrice = oi.UnitPrice * float64(oi.Quantity)
	return nil
}

// BeforeUpdate hook to recalculate total price
func (oi *OrderItem) BeforeUpdate(tx *gorm.DB) error {
	oi.TotalPrice = oi.UnitPrice * float64(oi.Quantity)
	return nil
}

// TableName returns the table name for OrderItem model
func (OrderItem) TableName() string {
	return "order_items"
}

// GetSubtotal returns the subtotal for this order item
func (oi *OrderItem) GetSubtotal() float64 {
	return oi.UnitPrice * float64(oi.Quantity)
}
