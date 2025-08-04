package workers

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// BaseJob provides common job functionality
type BaseJob struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Priority   int       `json:"priority"`
	CreatedAt  time.Time `json:"created_at"`
	retryCount int32     // atomic counter
	MaxRetries int       `json:"max_retries"`
}

// NewBaseJob creates a new base job
func NewBaseJob(jobType string, priority int, maxRetries int) *BaseJob {
	return &BaseJob{
		ID:         uuid.New().String(),
		Type:       jobType,
		Priority:   priority,
		CreatedAt:  time.Now(),
		MaxRetries: maxRetries,
	}
}

// GetID returns the job ID
func (b *BaseJob) GetID() string {
	return b.ID
}

// GetType returns the job type
func (b *BaseJob) GetType() string {
	return b.Type
}

// GetPriority returns the job priority
func (b *BaseJob) GetPriority() int {
	return b.Priority
}

// GetRetryCount returns the current retry count
func (b *BaseJob) GetRetryCount() int {
	return int(atomic.LoadInt32(&b.retryCount))
}

// IncrementRetryCount increments the retry counter
func (b *BaseJob) IncrementRetryCount() {
	atomic.AddInt32(&b.retryCount, 1)
}

// GetMaxRetries returns the maximum number of retries
func (b *BaseJob) GetMaxRetries() int {
	return b.MaxRetries
}

// Job type constants
const (
	JobTypeReportGeneration    = "report_generation"
	JobTypeNotification        = "notification"
	JobTypeAuditProcessing     = "audit_processing"
	JobTypeBulkProcessing      = "bulk_processing"
	JobTypeExternalIntegration = "external_integration"
	JobTypeCacheWarming        = "cache_warming"
	JobTypeCleanup             = "cleanup"
	JobTypeDataExport          = "data_export"
	JobTypePaymentRetry        = "payment_retry"
)

// Priority levels
const (
	PriorityLow      = 1
	PriorityNormal   = 5
	PriorityHigh     = 8
	PriorityCritical = 10
)

// ReportGenerationJob handles report generation tasks
type ReportGenerationJob struct {
	*BaseJob
	ReportType string                 `json:"report_type"`
	Parameters map[string]interface{} `json:"parameters"`
	OutputPath string                 `json:"output_path"`
	executor   ReportExecutor
}

// ReportExecutor interface for report generation
type ReportExecutor interface {
	GenerateReport(ctx context.Context, reportType string, params map[string]interface{}) (string, error)
}

// NewReportGenerationJob creates a new report generation job
func NewReportGenerationJob(reportType string, params map[string]interface{}, executor ReportExecutor) *ReportGenerationJob {
	return &ReportGenerationJob{
		BaseJob:    NewBaseJob(JobTypeReportGeneration, PriorityNormal, 3),
		ReportType: reportType,
		Parameters: params,
		executor:   executor,
	}
}

// Execute runs the report generation job
func (r *ReportGenerationJob) Execute(ctx context.Context) error {
	if r.executor == nil {
		return fmt.Errorf("no report executor configured")
	}

	outputPath, err := r.executor.GenerateReport(ctx, r.ReportType, r.Parameters)
	if err != nil {
		return fmt.Errorf("failed to generate report %s: %w", r.ReportType, err)
	}

	r.OutputPath = outputPath
	return nil
}

// NotificationJob handles notification sending
type NotificationJob struct {
	*BaseJob
	RecipientType    string                 `json:"recipient_type"`
	RecipientID      string                 `json:"recipient_id"`
	NotificationType string                 `json:"notification_type"`
	Template         string                 `json:"template"`
	Data             map[string]interface{} `json:"data"`
	executor         NotificationExecutor
}

// NotificationExecutor interface for sending notifications
type NotificationExecutor interface {
	SendNotification(ctx context.Context, recipient, notificationType, template string, data map[string]interface{}) error
}

// NewNotificationJob creates a new notification job
func NewNotificationJob(recipientType, recipientID, notificationType, template string, data map[string]interface{}, executor NotificationExecutor) *NotificationJob {
	priority := PriorityNormal
	if notificationType == "critical" || notificationType == "payment_failure" {
		priority = PriorityHigh
	}

	return &NotificationJob{
		BaseJob:          NewBaseJob(JobTypeNotification, priority, 5),
		RecipientType:    recipientType,
		RecipientID:      recipientID,
		NotificationType: notificationType,
		Template:         template,
		Data:             data,
		executor:         executor,
	}
}

// Execute runs the notification job
func (n *NotificationJob) Execute(ctx context.Context) error {
	if n.executor == nil {
		return fmt.Errorf("no notification executor configured")
	}

	return n.executor.SendNotification(ctx, n.RecipientID, n.NotificationType, n.Template, n.Data)
}

// AuditProcessingJob handles audit log processing
type AuditProcessingJob struct {
	*BaseJob
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Action     string                 `json:"action"`
	UserID     string                 `json:"user_id"`
	Changes    map[string]interface{} `json:"changes"`
	executor   AuditExecutor
}

// AuditExecutor interface for audit processing
type AuditExecutor interface {
	ProcessAuditLog(ctx context.Context, entityType, entityID, action, userID string, changes map[string]interface{}) error
}

// NewAuditProcessingJob creates a new audit processing job
func NewAuditProcessingJob(entityType, entityID, action, userID string, changes map[string]interface{}, executor AuditExecutor) *AuditProcessingJob {
	return &AuditProcessingJob{
		BaseJob:    NewBaseJob(JobTypeAuditProcessing, PriorityNormal, 3),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		UserID:     userID,
		Changes:    changes,
		executor:   executor,
	}
}

// Execute runs the audit processing job
func (a *AuditProcessingJob) Execute(ctx context.Context) error {
	if a.executor == nil {
		return fmt.Errorf("no audit executor configured")
	}

	return a.executor.ProcessAuditLog(ctx, a.EntityType, a.EntityID, a.Action, a.UserID, a.Changes)
}

// BulkProcessingJob handles bulk operations
type BulkProcessingJob struct {
	*BaseJob
	OperationType string                 `json:"operation_type"`
	EntityIDs     []string               `json:"entity_ids"`
	BatchSize     int                    `json:"batch_size"`
	Parameters    map[string]interface{} `json:"parameters"`
	executor      BulkExecutor
}

// BulkExecutor interface for bulk operations
type BulkExecutor interface {
	ProcessBulkOperation(ctx context.Context, operationType string, entityIDs []string, batchSize int, params map[string]interface{}) error
}

// NewBulkProcessingJob creates a new bulk processing job
func NewBulkProcessingJob(operationType string, entityIDs []string, batchSize int, params map[string]interface{}, executor BulkExecutor) *BulkProcessingJob {
	priority := PriorityNormal
	if len(entityIDs) > 1000 {
		priority = PriorityLow // Large bulk operations get lower priority
	}

	return &BulkProcessingJob{
		BaseJob:       NewBaseJob(JobTypeBulkProcessing, priority, 2),
		OperationType: operationType,
		EntityIDs:     entityIDs,
		BatchSize:     batchSize,
		Parameters:    params,
		executor:      executor,
	}
}

// Execute runs the bulk processing job
func (b *BulkProcessingJob) Execute(ctx context.Context) error {
	if b.executor == nil {
		return fmt.Errorf("no bulk executor configured")
	}

	return b.executor.ProcessBulkOperation(ctx, b.OperationType, b.EntityIDs, b.BatchSize, b.Parameters)
}

// ExternalIntegrationJob handles external API calls
type ExternalIntegrationJob struct {
	*BaseJob
	ServiceName string                 `json:"service_name"`
	Operation   string                 `json:"operation"`
	Payload     map[string]interface{} `json:"payload"`
	Timeout     time.Duration          `json:"timeout"`
	executor    IntegrationExecutor
}

// IntegrationExecutor interface for external integrations
type IntegrationExecutor interface {
	ExecuteIntegration(ctx context.Context, serviceName, operation string, payload map[string]interface{}) (map[string]interface{}, error)
}

// NewExternalIntegrationJob creates a new external integration job
func NewExternalIntegrationJob(serviceName, operation string, payload map[string]interface{}, timeout time.Duration, executor IntegrationExecutor) *ExternalIntegrationJob {
	return &ExternalIntegrationJob{
		BaseJob:     NewBaseJob(JobTypeExternalIntegration, PriorityHigh, 3),
		ServiceName: serviceName,
		Operation:   operation,
		Payload:     payload,
		Timeout:     timeout,
		executor:    executor,
	}
}

// Execute runs the external integration job
func (e *ExternalIntegrationJob) Execute(ctx context.Context) error {
	if e.executor == nil {
		return fmt.Errorf("no integration executor configured")
	}

	// Apply timeout if specified
	if e.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}

	_, err := e.executor.ExecuteIntegration(ctx, e.ServiceName, e.Operation, e.Payload)
	return err
}
