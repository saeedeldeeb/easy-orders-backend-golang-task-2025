package notifications

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType defines the type of notification
type NotificationType string

const (
	NotificationTypeOrderConfirmation NotificationType = "order_confirmation"
	NotificationTypeOrderShipped      NotificationType = "order_shipped"
	NotificationTypeOrderDelivered    NotificationType = "order_delivered"
	NotificationTypeOrderCancelled    NotificationType = "order_cancelled"
	NotificationTypePaymentSuccess    NotificationType = "payment_success"
	NotificationTypePaymentFailed     NotificationType = "payment_failed"
	NotificationTypeLowStock          NotificationType = "low_stock"
	NotificationTypeWelcome           NotificationType = "welcome"
	NotificationTypePasswordReset     NotificationType = "password_reset"
	NotificationTypePromotion         NotificationType = "promotion"
	NotificationTypeSystemAlert       NotificationType = "system_alert"
)

// NotificationPriority defines the priority level
type NotificationPriority int

const (
	PriorityLow      NotificationPriority = 1
	PriorityNormal   NotificationPriority = 5
	PriorityHigh     NotificationPriority = 8
	PriorityCritical NotificationPriority = 10
)

// NotificationStatus represents the delivery status
type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusSending   NotificationStatus = "sending"
	StatusSent      NotificationStatus = "sent"
	StatusFailed    NotificationStatus = "failed"
	StatusRetrying  NotificationStatus = "retrying"
	StatusCancelled NotificationStatus = "cancelled"
)

// Notification represents a notification message
type Notification struct {
	ID         string                 `json:"id"`
	Type       NotificationType       `json:"type"`
	Priority   NotificationPriority   `json:"priority"`
	Status     NotificationStatus     `json:"status"`
	Channel    string                 `json:"channel"`
	Recipient  string                 `json:"recipient"`
	Subject    string                 `json:"subject"`
	Body       string                 `json:"body"`
	Data       map[string]interface{} `json:"data"`
	TemplateID string                 `json:"template_id,omitempty"`
	Metadata   map[string]string      `json:"metadata"`

	// Retry configuration
	MaxRetries int           `json:"max_retries"`
	RetryCount int           `json:"retry_count"`
	RetryDelay time.Duration `json:"retry_delay"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	FailedAt    *time.Time `json:"failed_at,omitempty"`

	// Error tracking
	LastError    string              `json:"last_error,omitempty"`
	ErrorHistory []NotificationError `json:"error_history,omitempty"`
}

// NotificationError represents an error that occurred during notification processing
type NotificationError struct {
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
	Attempt   int       `json:"attempt"`
	Channel   string    `json:"channel"`
}

// NewNotification creates a new notification
func NewNotification(notificationType NotificationType, channel, recipient string) *Notification {
	return &Notification{
		ID:         uuid.New().String(),
		Type:       notificationType,
		Priority:   PriorityNormal,
		Status:     StatusPending,
		Channel:    channel,
		Recipient:  recipient,
		Data:       make(map[string]interface{}),
		Metadata:   make(map[string]string),
		MaxRetries: 3,
		RetryDelay: time.Minute,
		CreatedAt:  time.Now(),
	}
}

// MarkAsSending updates the notification status to sending
func (n *Notification) MarkAsSending() {
	n.Status = StatusSending
}

// MarkAsSent updates the notification status to sent
func (n *Notification) MarkAsSent() {
	n.Status = StatusSent
	now := time.Now()
	n.SentAt = &now
}

// MarkAsFailed updates the notification status to failed
func (n *Notification) MarkAsFailed(err error) {
	n.Status = StatusFailed
	n.LastError = err.Error()
	now := time.Now()
	n.FailedAt = &now

	// Add to error history
	n.ErrorHistory = append(n.ErrorHistory, NotificationError{
		Error:     err.Error(),
		Timestamp: now,
		Attempt:   n.RetryCount + 1,
		Channel:   n.Channel,
	})
}

// MarkAsRetrying updates the notification status to retrying
func (n *Notification) MarkAsRetrying() {
	n.Status = StatusRetrying
	n.RetryCount++
}

// ShouldRetry determines if the notification should be retried
func (n *Notification) ShouldRetry() bool {
	return n.Status == StatusFailed && n.RetryCount < n.MaxRetries
}

// GetNextRetryTime calculates when the notification should be retried
func (n *Notification) GetNextRetryTime() time.Time {
	// Exponential backoff: delay * (2 ^ retry_count)
	backoffDelay := n.RetryDelay * time.Duration(1<<n.RetryCount)

	// Cap at 1 hour
	if backoffDelay > time.Hour {
		backoffDelay = time.Hour
	}

	return time.Now().Add(backoffDelay)
}

// IsExpired checks if the notification has expired (older than 24 hours)
func (n *Notification) IsExpired() bool {
	return time.Since(n.CreatedAt) > 24*time.Hour
}

// IsScheduled checks if the notification is scheduled for future delivery
func (n *Notification) IsScheduled() bool {
	return n.ScheduledAt != nil && n.ScheduledAt.After(time.Now())
}

// IsReadyToSend checks if the notification is ready to be sent
func (n *Notification) IsReadyToSend() bool {
	if n.IsExpired() {
		return false
	}

	if n.IsScheduled() {
		return false
	}

	return n.Status == StatusPending || (n.Status == StatusRetrying && time.Now().After(n.GetNextRetryTime()))
}

// SetPriority sets the notification priority and adjusts retry behavior
func (n *Notification) SetPriority(priority NotificationPriority) {
	n.Priority = priority

	// Adjust retry behavior based on priority
	switch priority {
	case PriorityCritical:
		n.MaxRetries = 5
		n.RetryDelay = 30 * time.Second
	case PriorityHigh:
		n.MaxRetries = 4
		n.RetryDelay = time.Minute
	case PriorityNormal:
		n.MaxRetries = 3
		n.RetryDelay = time.Minute
	case PriorityLow:
		n.MaxRetries = 2
		n.RetryDelay = 5 * time.Minute
	}
}

// GetPriorityScore returns a numeric score for priority-based sorting
func (n *Notification) GetPriorityScore() int {
	score := int(n.Priority) * 100

	// Add urgency based on creation time (older = more urgent)
	age := time.Since(n.CreatedAt)
	urgency := int(age.Minutes())

	return score + urgency
}

// Clone creates a deep copy of the notification
func (n *Notification) Clone() *Notification {
	clone := *n

	// Deep copy maps
	clone.Data = make(map[string]interface{})
	for k, v := range n.Data {
		clone.Data[k] = v
	}

	clone.Metadata = make(map[string]string)
	for k, v := range n.Metadata {
		clone.Metadata[k] = v
	}

	// Deep copy error history
	clone.ErrorHistory = make([]NotificationError, len(n.ErrorHistory))
	copy(clone.ErrorHistory, n.ErrorHistory)

	return &clone
}
