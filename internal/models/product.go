package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Product represents a product in the system
type Product struct {
	ID          string         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"not null;size:255;index" json:"name" validate:"required,min=1,max=255"`
	Description string         `gorm:"type:text" json:"description"`
	Price       float64        `gorm:"type:decimal(10,2);not null" json:"price" validate:"required,gt=0"`
	SKU         string         `gorm:"uniqueIndex;not null;size:100" json:"sku" validate:"required"`
	CategoryID  *string        `gorm:"type:uuid;index" json:"category_id"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Inventory  *Inventory  `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"inventory,omitempty"`
	OrderItems []OrderItem `gorm:"foreignKey:ProductID;constraint:OnDelete:RESTRICT" json:"order_items,omitempty"`
}

// BeforeCreate hook to generate UUID if not provided
func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// TableName returns the table name for Product model
func (Product) TableName() string {
	return "products"
}

// IsAvailable returns true if product is active and has inventory
func (p *Product) IsAvailable() bool {
	return p.IsActive && p.Inventory != nil && p.Inventory.Available > 0
}

// GetAvailableStock returns the available stock quantity
func (p *Product) GetAvailableStock() int {
	if p.Inventory == nil {
		return 0
	}
	return p.Inventory.Available
}
