package notifications

import (
	"context"
	"fmt"
	"sync"
	"time"

	"easy-orders-backend/pkg/logger"
)

// NotificationChannel represents a delivery method for notifications
type NotificationChannel interface {
	// Send delivers a notification through this channel
	Send(ctx context.Context, notification *Notification) error

	// GetName returns the channel name
	GetName() string

	// IsEnabled returns whether this channel is currently enabled
	IsEnabled() bool

	// GetRateLimit returns the rate limit for this channel (messages per second)
	GetRateLimit() int

	// SupportsTemplate returns whether this channel supports templating
	SupportsTemplate() bool
}

// NotificationProvider manages multiple notification channels
type NotificationProvider struct {
	channels map[string]NotificationChannel
	logger   *logger.Logger
	mu       sync.RWMutex
}

// NewNotificationProvider creates a new notification provider
func NewNotificationProvider(logger *logger.Logger) *NotificationProvider {
	return &NotificationProvider{
		channels: make(map[string]NotificationChannel),
		logger:   logger,
	}
}

// RegisterChannel registers a new notification channel
func (np *NotificationProvider) RegisterChannel(channel NotificationChannel) {
	np.mu.Lock()
	defer np.mu.Unlock()

	np.channels[channel.GetName()] = channel
	np.logger.Info("Notification channel registered", "channel", channel.GetName())
}

// GetChannel retrieves a notification channel by name
func (np *NotificationProvider) GetChannel(name string) (NotificationChannel, bool) {
	np.mu.RLock()
	defer np.mu.RUnlock()

	channel, exists := np.channels[name]
	return channel, exists
}

// GetAvailableChannels returns all enabled channels
func (np *NotificationProvider) GetAvailableChannels() []NotificationChannel {
	np.mu.RLock()
	defer np.mu.RUnlock()

	var channels []NotificationChannel
	for _, channel := range np.channels {
		if channel.IsEnabled() {
			channels = append(channels, channel)
		}
	}

	return channels
}

// SendNotification sends a notification through the specified channel
func (np *NotificationProvider) SendNotification(ctx context.Context, channelName string, notification *Notification) error {
	channel, exists := np.GetChannel(channelName)
	if !exists {
		return fmt.Errorf("notification channel %s not found", channelName)
	}

	if !channel.IsEnabled() {
		return fmt.Errorf("notification channel %s is disabled", channelName)
	}

	return channel.Send(ctx, notification)
}

// EmailChannel implements email notifications
type EmailChannel struct {
	name       string
	enabled    bool
	rateLimit  int
	smtpConfig SMTPConfig
	logger     *logger.Logger
}

// SMTPConfig contains email server configuration
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	UseTLS   bool   `json:"use_tls"`
}

// NewEmailChannel creates a new email notification channel
func NewEmailChannel(config SMTPConfig, logger *logger.Logger) *EmailChannel {
	return &EmailChannel{
		name:       "email",
		enabled:    true,
		rateLimit:  10, // 10 emails per second
		smtpConfig: config,
		logger:     logger,
	}
}

func (e *EmailChannel) Send(ctx context.Context, notification *Notification) error {
	e.logger.Debug("Sending email notification",
		"recipient", notification.Recipient,
		"subject", notification.Subject)

	// Simulate email sending
	// In a real implementation, this would use SMTP or an email service
	time.Sleep(100 * time.Millisecond) // Simulate network delay

	e.logger.Info("Email notification sent",
		"recipient", notification.Recipient,
		"subject", notification.Subject)

	return nil
}

func (e *EmailChannel) GetName() string        { return e.name }
func (e *EmailChannel) IsEnabled() bool        { return e.enabled }
func (e *EmailChannel) GetRateLimit() int      { return e.rateLimit }
func (e *EmailChannel) SupportsTemplate() bool { return true }

// SMSChannel implements SMS notifications
type SMSChannel struct {
	name      string
	enabled   bool
	rateLimit int
	apiKey    string
	apiURL    string
	logger    *logger.Logger
}

// NewSMSChannel creates a new SMS notification channel
func NewSMSChannel(apiKey, apiURL string, logger *logger.Logger) *SMSChannel {
	return &SMSChannel{
		name:      "sms",
		enabled:   true,
		rateLimit: 5, // 5 SMS per second
		apiKey:    apiKey,
		apiURL:    apiURL,
		logger:    logger,
	}
}

func (s *SMSChannel) Send(ctx context.Context, notification *Notification) error {
	s.logger.Debug("Sending SMS notification",
		"recipient", notification.Recipient,
		"message_length", len(notification.Body))

	// Simulate SMS sending
	time.Sleep(200 * time.Millisecond) // Simulate API call

	s.logger.Info("SMS notification sent",
		"recipient", notification.Recipient)

	return nil
}

func (s *SMSChannel) GetName() string        { return s.name }
func (s *SMSChannel) IsEnabled() bool        { return s.enabled }
func (s *SMSChannel) GetRateLimit() int      { return s.rateLimit }
func (s *SMSChannel) SupportsTemplate() bool { return false }

// PushChannel implements push notifications
type PushChannel struct {
	name      string
	enabled   bool
	rateLimit int
	logger    *logger.Logger
}

// NewPushChannel creates a new push notification channel
func NewPushChannel(logger *logger.Logger) *PushChannel {
	return &PushChannel{
		name:      "push",
		enabled:   true,
		rateLimit: 50, // 50 push notifications per second
		logger:    logger,
	}
}

func (p *PushChannel) Send(ctx context.Context, notification *Notification) error {
	p.logger.Debug("Sending push notification",
		"recipient", notification.Recipient,
		"title", notification.Subject)

	// Simulate push notification sending
	time.Sleep(50 * time.Millisecond) // Simulate API call

	p.logger.Info("Push notification sent",
		"recipient", notification.Recipient)

	return nil
}

func (p *PushChannel) GetName() string        { return p.name }
func (p *PushChannel) IsEnabled() bool        { return p.enabled }
func (p *PushChannel) GetRateLimit() int      { return p.rateLimit }
func (p *PushChannel) SupportsTemplate() bool { return true }

// WebhookChannel implements webhook notifications
type WebhookChannel struct {
	name       string
	enabled    bool
	rateLimit  int
	webhookURL string
	logger     *logger.Logger
}

// NewWebhookChannel creates a new webhook notification channel
func NewWebhookChannel(webhookURL string, logger *logger.Logger) *WebhookChannel {
	return &WebhookChannel{
		name:       "webhook",
		enabled:    true,
		rateLimit:  20, // 20 webhook calls per second
		webhookURL: webhookURL,
		logger:     logger,
	}
}

func (w *WebhookChannel) Send(ctx context.Context, notification *Notification) error {
	w.logger.Debug("Sending webhook notification",
		"recipient", notification.Recipient,
		"webhook_url", w.webhookURL)

	// Simulate webhook call
	time.Sleep(150 * time.Millisecond) // Simulate HTTP request

	w.logger.Info("Webhook notification sent",
		"recipient", notification.Recipient,
		"webhook_url", w.webhookURL)

	return nil
}

func (w *WebhookChannel) GetName() string        { return w.name }
func (w *WebhookChannel) IsEnabled() bool        { return w.enabled }
func (w *WebhookChannel) GetRateLimit() int      { return w.rateLimit }
func (w *WebhookChannel) SupportsTemplate() bool { return true }
