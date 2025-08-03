package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRole defines user roles in the system
type UserRole string

const (
	UserRoleCustomer UserRole = "customer"
	UserRoleAdmin    UserRole = "admin"
)

// User represents a user in the system (customers and admins)
type User struct {
	ID        string         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email     string         `gorm:"uniqueIndex;not null;size:255" json:"email" validate:"required,email"`
	Password  string         `gorm:"not null;size:255" json:"-"` // Never expose password in JSON
	Name      string         `gorm:"not null;size:100" json:"name" validate:"required,min=2,max=100"`
	Role      UserRole       `gorm:"type:varchar(20);not null;default:'customer'" json:"role"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Orders        []Order        `gorm:"foreignKey:UserID;constraint:OnDelete:RESTRICT" json:"orders,omitempty"`
	Notifications []Notification `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"notifications,omitempty"`
	AuditLogs     []AuditLog     `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"audit_logs,omitempty"`
}

// BeforeCreate hook to generate UUID if not provided
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// TableName returns the table name for User model
func (User) TableName() string {
	return "users"
}

// IsAdmin returns true if user is an admin
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// IsCustomer returns true if user is a customer
func (u *User) IsCustomer() bool {
	return u.Role == UserRoleCustomer
}
