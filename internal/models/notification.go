package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationType defines different types of notifications
type NotificationType string

const (
	NotificationTypeOrderConfirmed NotificationType = "order_confirmed"
	NotificationTypeOrderShipped   NotificationType = "order_shipped"
	NotificationTypeOrderDelivered NotificationType = "order_delivered"
	NotificationTypeOrderCancelled NotificationType = "order_cancelled"
	NotificationTypePaymentSuccess NotificationType = "payment_success"
	NotificationTypePaymentFailed  NotificationType = "payment_failed"
	NotificationTypeLowStock       NotificationType = "low_stock"
	NotificationTypePromotion      NotificationType = "promotion"
	NotificationTypeSystem         NotificationType = "system"
)

// NotificationChannel defines how the notification should be sent
type NotificationChannel string

const (
	NotificationChannelEmail NotificationChannel = "email"
	NotificationChannelSMS   NotificationChannel = "sms"
	NotificationChannelPush  NotificationChannel = "push"
	NotificationChannelInApp NotificationChannel = "in_app"
)

// Notification represents system notifications
type Notification struct {
	ID        string              `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    string              `gorm:"type:uuid;not null;index" json:"user_id" validate:"required"`
	Type      NotificationType    `gorm:"type:varchar(30);not null;index" json:"type" validate:"required"`
	Channel   NotificationChannel `gorm:"type:varchar(20);not null" json:"channel" validate:"required"`
	Title     string              `gorm:"not null;size:255" json:"title" validate:"required,max=255"`
	Body      string              `gorm:"type:text;not null" json:"body" validate:"required"`
	Data      string              `gorm:"type:jsonb" json:"data"` // Additional data as JSON
	Read      bool                `gorm:"default:false;index" json:"read"`
	ReadAt    *time.Time          `json:"read_at"`
	SentAt    *time.Time          `json:"sent_at"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// BeforeCreate hook to generate UUID if not provided
func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return nil
}

// TableName returns the table name for Notification model
func (Notification) TableName() string {
	return "notifications"
}

// MarkAsRead marks the notification as read
func (n *Notification) MarkAsRead() {
	if !n.Read {
		n.Read = true
		now := time.Now()
		n.ReadAt = &now
	}
}

// MarkAsSent marks the notification as sent
func (n *Notification) MarkAsSent() {
	if n.SentAt == nil {
		now := time.Now()
		n.SentAt = &now
	}
}

// IsOrderRelated returns true if notification is order-related
func (n *Notification) IsOrderRelated() bool {
	return n.Type == NotificationTypeOrderConfirmed ||
		n.Type == NotificationTypeOrderShipped ||
		n.Type == NotificationTypeOrderDelivered ||
		n.Type == NotificationTypeOrderCancelled
}

// IsPaymentRelated returns true if notification is payment-related
func (n *Notification) IsPaymentRelated() bool {
	return n.Type == NotificationTypePaymentSuccess ||
		n.Type == NotificationTypePaymentFailed
}
