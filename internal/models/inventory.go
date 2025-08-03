package models

import (
	"time"

	"gorm.io/gorm"
)

// Inventory represents stock management for products
type Inventory struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductID string    `gorm:"type:uuid;uniqueIndex;not null" json:"product_id"`
	Quantity  int       `gorm:"not null;default:0" json:"quantity" validate:"gte=0"`
	Reserved  int       `gorm:"not null;default:0" json:"reserved" validate:"gte=0"`
	Available int       `gorm:"not null;default:0" json:"available" validate:"gte=0"`
	MinStock  int       `gorm:"not null;default:10" json:"min_stock" validate:"gte=0"`
	MaxStock  int       `gorm:"not null;default:1000" json:"max_stock" validate:"gt=0"`
	Version   int       `gorm:"not null;default:1" json:"version"` // For optimistic locking
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Product *Product `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"product,omitempty"`
}

// TableName returns the table name for Inventory model
func (Inventory) TableName() string {
	return "inventory"
}

// BeforeUpdate hook to update available quantity
func (i *Inventory) BeforeUpdate(tx *gorm.DB) error {
	i.Available = i.Quantity - i.Reserved
	return nil
}

// BeforeCreate hook to set available quantity
func (i *Inventory) BeforeCreate(tx *gorm.DB) error {
	i.Available = i.Quantity - i.Reserved
	return nil
}

// IsLowStock returns true if current stock is below minimum threshold
func (i *Inventory) IsLowStock() bool {
	return i.Available <= i.MinStock
}

// CanReserve checks if the requested quantity can be reserved
func (i *Inventory) CanReserve(quantity int) bool {
	return i.Available >= quantity && quantity > 0
}

// Reserve reserves the specified quantity
func (i *Inventory) Reserve(quantity int) error {
	if !i.CanReserve(quantity) {
		return gorm.ErrInvalidData
	}
	i.Reserved += quantity
	i.Available = i.Quantity - i.Reserved
	return nil
}

// Release releases the specified reserved quantity
func (i *Inventory) Release(quantity int) error {
	if i.Reserved < quantity || quantity <= 0 {
		return gorm.ErrInvalidData
	}
	i.Reserved -= quantity
	i.Available = i.Quantity - i.Reserved
	return nil
}

// Fulfill confirms the reserved quantity and reduces total stock
func (i *Inventory) Fulfill(quantity int) error {
	if i.Reserved < quantity || quantity <= 0 {
		return gorm.ErrInvalidData
	}
	i.Reserved -= quantity
	i.Quantity -= quantity
	i.Available = i.Quantity - i.Reserved
	return nil
}
