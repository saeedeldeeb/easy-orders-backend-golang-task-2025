package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditAction defines the type of action performed
type AuditAction string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
	AuditActionLogin  AuditAction = "login"
	AuditActionLogout AuditAction = "logout"
	AuditActionAccess AuditAction = "access"
)

// AuditLog represents audit trail for tracking changes
type AuditLog struct {
	ID         string      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID     *string     `gorm:"type:uuid;index" json:"user_id"` // Nullable for system actions
	EntityType string      `gorm:"not null;size:50;index" json:"entity_type" validate:"required"`
	EntityID   string      `gorm:"not null;size:255;index" json:"entity_id" validate:"required"`
	Action     AuditAction `gorm:"type:varchar(20);not null;index" json:"action" validate:"required"`
	OldValues  string      `gorm:"type:jsonb" json:"old_values"` // JSON representation of old values
	NewValues  string      `gorm:"type:jsonb" json:"new_values"` // JSON representation of new values
	IPAddress  string      `gorm:"size:45" json:"ip_address"`    // IPv4 or IPv6
	UserAgent  string      `gorm:"size:500" json:"user_agent"`
	CreatedAt  time.Time   `json:"created_at"`

	// Relationships
	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"user,omitempty"`
}

// BeforeCreate hook to generate UUID if not provided
func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == "" {
		al.ID = uuid.New().String()
	}
	return nil
}

// TableName returns the table name for AuditLog model
func (AuditLog) TableName() string {
	return "audit_logs"
}

// SetOldValues sets the old values as JSON string
func (al *AuditLog) SetOldValues(values interface{}) error {
	if values == nil {
		al.OldValues = ""
		return nil
	}
	jsonData, err := json.Marshal(values)
	if err != nil {
		return err
	}
	al.OldValues = string(jsonData)
	return nil
}

// SetNewValues sets the new values as JSON string
func (al *AuditLog) SetNewValues(values interface{}) error {
	if values == nil {
		al.NewValues = ""
		return nil
	}
	jsonData, err := json.Marshal(values)
	if err != nil {
		return err
	}
	al.NewValues = string(jsonData)
	return nil
}

// GetOldValues parses old values from JSON string
func (al *AuditLog) GetOldValues(target interface{}) error {
	if al.OldValues == "" {
		return nil
	}
	return json.Unmarshal([]byte(al.OldValues), target)
}

// GetNewValues parses new values from JSON string
func (al *AuditLog) GetNewValues(target interface{}) error {
	if al.NewValues == "" {
		return nil
	}
	return json.Unmarshal([]byte(al.NewValues), target)
}

// IsSystemAction returns true if this is a system-generated action
func (al *AuditLog) IsSystemAction() bool {
	return al.UserID == nil
}

// IsUserAction returns true if this is a user-generated action
func (al *AuditLog) IsUserAction() bool {
	return al.UserID != nil
}
