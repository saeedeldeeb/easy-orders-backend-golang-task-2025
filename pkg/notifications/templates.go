package notifications

import (
	"bytes"
	"fmt"
	"html/template"
	"sync"

	"easy-orders-backend/pkg/logger"
)

// Template represents a notification template
type Template struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      NotificationType  `json:"type"`
	Channel   string            `json:"channel"`
	Subject   string            `json:"subject"`
	Body      string            `json:"body"`
	Variables []string          `json:"variables"`
	Metadata  map[string]string `json:"metadata"`
	IsActive  bool              `json:"is_active"`
}

// TemplateManager manages notification templates
type TemplateManager struct {
	templates map[string]*Template
	compiled  map[string]*template.Template
	mutex     sync.RWMutex
	logger    *logger.Logger
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(logger *logger.Logger) *TemplateManager {
	tm := &TemplateManager{
		templates: make(map[string]*Template),
		compiled:  make(map[string]*template.Template),
		logger:    logger,
	}

	// Load default templates
	tm.loadDefaultTemplates()

	return tm
}

// RegisterTemplate registers a new template
func (tm *TemplateManager) RegisterTemplate(tmpl *Template) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Compile the template
	compiled, err := template.New(tmpl.ID).Parse(tmpl.Body)
	if err != nil {
		return fmt.Errorf("failed to compile template %s: %w", tmpl.ID, err)
	}

	tm.templates[tmpl.ID] = tmpl
	tm.compiled[tmpl.ID] = compiled

	tm.logger.Info("Template registered", "template_id", tmpl.ID, "type", tmpl.Type)
	return nil
}

// GetTemplate retrieves a template by ID
func (tm *TemplateManager) GetTemplate(id string) (*Template, bool) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	tmpl, exists := tm.templates[id]
	return tmpl, exists
}

// ApplyTemplate applies a template to notification data
func (tm *TemplateManager) ApplyTemplate(templateID string, data map[string]interface{}) (subject, body string, err error) {
	tm.mutex.RLock()
	tmpl, exists := tm.templates[templateID]
	compiled, compiledExists := tm.compiled[templateID]
	tm.mutex.RUnlock()

	if !exists || !compiledExists {
		return "", "", fmt.Errorf("template %s not found", templateID)
	}

	if !tmpl.IsActive {
		return "", "", fmt.Errorf("template %s is inactive", templateID)
	}

	// Apply subject template
	subjectTmpl, err := template.New("subject").Parse(tmpl.Subject)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse subject template: %w", err)
	}

	var subjectBuf bytes.Buffer
	if err := subjectTmpl.Execute(&subjectBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute subject template: %w", err)
	}

	// Apply body template
	var bodyBuf bytes.Buffer
	if err := compiled.Execute(&bodyBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute body template: %w", err)
	}

	return subjectBuf.String(), bodyBuf.String(), nil
}

// ListTemplates returns all registered templates
func (tm *TemplateManager) ListTemplates() []*Template {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	templates := make([]*Template, 0, len(tm.templates))
	for _, tmpl := range tm.templates {
		templates = append(templates, tmpl)
	}

	return templates
}

// GetTemplatesByType returns templates for a specific notification type
func (tm *TemplateManager) GetTemplatesByType(notificationType NotificationType) []*Template {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	var templates []*Template
	for _, tmpl := range tm.templates {
		if tmpl.Type == notificationType && tmpl.IsActive {
			templates = append(templates, tmpl)
		}
	}

	return templates
}

// loadDefaultTemplates loads predefined templates
func (tm *TemplateManager) loadDefaultTemplates() {
	templates := []*Template{
		{
			ID:      "order_confirmation_email",
			Name:    "Order Confirmation Email",
			Type:    NotificationTypeOrderConfirmation,
			Channel: "email",
			Subject: "Order Confirmation - #{{.OrderID}}",
			Body: `
Dear {{.CustomerName}},

Thank you for your order! We're excited to get your items to you.

Order Details:
- Order ID: {{.OrderID}}
- Total Amount: ${{.TotalAmount}}
- Order Date: {{.OrderDate}}

Items Ordered:
{{range .Items}}
- {{.Name}} (Qty: {{.Quantity}}) - ${{.Price}}
{{end}}

Shipping Address:
{{.ShippingAddress}}

We'll send you another email when your order ships.

Best regards,
The Easy Orders Team
			`,
			Variables: []string{"CustomerName", "OrderID", "TotalAmount", "OrderDate", "Items", "ShippingAddress"},
			IsActive:  true,
		},
		{
			ID:      "order_shipped_email",
			Name:    "Order Shipped Email",
			Type:    NotificationTypeOrderShipped,
			Channel: "email",
			Subject: "Your Order Has Shipped - #{{.OrderID}}",
			Body: `
Dear {{.CustomerName}},

Great news! Your order is on its way.

Order Details:
- Order ID: {{.OrderID}}
- Tracking Number: {{.TrackingNumber}}
- Carrier: {{.Carrier}}
- Estimated Delivery: {{.EstimatedDelivery}}

You can track your package at: {{.TrackingURL}}

Best regards,
The Easy Orders Team
			`,
			Variables: []string{"CustomerName", "OrderID", "TrackingNumber", "Carrier", "EstimatedDelivery", "TrackingURL"},
			IsActive:  true,
		},
		{
			ID:      "payment_success_email",
			Name:    "Payment Success Email",
			Type:    NotificationTypePaymentSuccess,
			Channel: "email",
			Subject: "Payment Confirmation - ${{.Amount}}",
			Body: `
Dear {{.CustomerName}},

Your payment has been processed successfully.

Payment Details:
- Amount: ${{.Amount}}
- Payment Method: {{.PaymentMethod}}
- Transaction ID: {{.TransactionID}}
- Date: {{.PaymentDate}}

Thank you for your business!

Best regards,
The Easy Orders Team
			`,
			Variables: []string{"CustomerName", "Amount", "PaymentMethod", "TransactionID", "PaymentDate"},
			IsActive:  true,
		},
		{
			ID:      "welcome_email",
			Name:    "Welcome Email",
			Type:    NotificationTypeWelcome,
			Channel: "email",
			Subject: "Welcome to Easy Orders, {{.CustomerName}}!",
			Body: `
Dear {{.CustomerName}},

Welcome to Easy Orders! We're thrilled to have you join our community.

Here's what you can do next:
- Browse our extensive product catalog
- Set up your profile and preferences
- Subscribe to our newsletter for exclusive deals

If you have any questions, our support team is here to help.

Happy shopping!

Best regards,
The Easy Orders Team
			`,
			Variables: []string{"CustomerName"},
			IsActive:  true,
		},
		{
			ID:        "low_stock_sms",
			Name:      "Low Stock SMS Alert",
			Type:      NotificationTypeLowStock,
			Channel:   "sms",
			Subject:   "Low Stock Alert",
			Body:      "ALERT: {{.ProductName}} is running low ({{.CurrentStock}} remaining). Restock needed urgently.",
			Variables: []string{"ProductName", "CurrentStock"},
			IsActive:  true,
		},
		{
			ID:        "order_confirmation_push",
			Name:      "Order Confirmation Push",
			Type:      NotificationTypeOrderConfirmation,
			Channel:   "push",
			Subject:   "Order Confirmed!",
			Body:      "Your order #{{.OrderID}} for ${{.TotalAmount}} has been confirmed. Thanks for shopping with us!",
			Variables: []string{"OrderID", "TotalAmount"},
			IsActive:  true,
		},
	}

	for _, tmpl := range templates {
		if err := tm.RegisterTemplate(tmpl); err != nil {
			tm.logger.Error("Failed to register default template", "template_id", tmpl.ID, "error", err)
		}
	}

	tm.logger.Info("Default templates loaded", "count", len(templates))
}
