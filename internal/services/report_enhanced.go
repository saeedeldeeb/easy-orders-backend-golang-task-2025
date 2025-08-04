package services

import (
	"context"
	"fmt"
	"time"

	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/logger"
	"easy-orders-backend/pkg/reports"
	"easy-orders-backend/pkg/workers"

	"github.com/google/uuid"
)

// EnhancedReportService provides concurrent report generation capabilities
type EnhancedReportService struct {
	reportManager *reports.ReportManager
	poolManager   *workers.PoolManager
	logger        *logger.Logger

	// Generators
	salesGenerator     *reports.SalesReportGenerator
	inventoryGenerator *reports.InventoryReportGenerator
	customerGenerator  *reports.CustomerReportGenerator
}

// NewEnhancedReportService creates a new enhanced report service
func NewEnhancedReportService(
	orderRepo repository.OrderRepository,
	paymentRepo repository.PaymentRepository,
	userRepo repository.UserRepository,
	productRepo repository.ProductRepository,
	inventoryRepo repository.InventoryRepository,
	poolManager *workers.PoolManager,
	logger *logger.Logger,
) *EnhancedReportService {
	// Create report manager
	reportManager := reports.NewReportManager(reports.DefaultReportManagerConfig(), logger)

	// Create generators
	salesGenerator := reports.NewSalesReportGenerator(orderRepo, paymentRepo, userRepo, productRepo, logger)
	inventoryGenerator := reports.NewInventoryReportGenerator(inventoryRepo, productRepo, orderRepo, logger)
	customerGenerator := reports.NewCustomerReportGenerator(userRepo, orderRepo, paymentRepo, logger)

	// Register generators
	reportManager.RegisterGenerator(salesGenerator)
	reportManager.RegisterGenerator(inventoryGenerator)
	reportManager.RegisterGenerator(customerGenerator)

	service := &EnhancedReportService{
		reportManager:      reportManager,
		poolManager:        poolManager,
		logger:             logger,
		salesGenerator:     salesGenerator,
		inventoryGenerator: inventoryGenerator,
		customerGenerator:  customerGenerator,
	}

	return service
}

// GenerateReportAsync generates a report asynchronously without blocking
func (ers *EnhancedReportService) GenerateReportAsync(ctx context.Context, reportType string, format string, parameters map[string]interface{}) (*reports.ReportResult, error) {
	ers.logger.Info("Generating report asynchronously",
		"type", reportType,
		"format", format)

	// Create report request
	req := &reports.ReportRequest{
		ID:         uuid.New().String(),
		Type:       reports.ReportType(reportType),
		Format:     reports.ReportFormat(format),
		Priority:   reports.ReportPriorityNormal,
		Parameters: parameters,
		CreatedAt:  time.Now(),
	}

	// Set expiration time (24 hours for most reports, 4 hours for inventory)
	var expiresAt time.Time
	if reportType == string(reports.ReportTypeLowStock) || reportType == string(reports.ReportTypeInventoryValue) {
		expiresAt = time.Now().Add(4 * time.Hour)
	} else {
		expiresAt = time.Now().Add(24 * time.Hour)
	}
	req.ExpiresAt = &expiresAt

	// Generate report asynchronously
	result, err := ers.reportManager.GenerateReportAsync(ctx, req)
	if err != nil {
		ers.logger.Error("Failed to start async report generation", "error", err, "type", reportType)
		return nil, fmt.Errorf("failed to start async report generation: %w", err)
	}

	ers.logger.Info("Async report generation started",
		"request_id", req.ID,
		"result_id", result.ID,
		"type", reportType)

	return result, nil
}

// GenerateReportSync generates a report synchronously
func (ers *EnhancedReportService) GenerateReportSync(ctx context.Context, reportType string, format string, parameters map[string]interface{}) (*reports.ReportResult, error) {
	ers.logger.Info("Generating report synchronously",
		"type", reportType,
		"format", format)

	// Create report request
	req := &reports.ReportRequest{
		ID:         uuid.New().String(),
		Type:       reports.ReportType(reportType),
		Format:     reports.ReportFormat(format),
		Priority:   reports.ReportPriorityHigh, // Higher priority for sync requests
		Parameters: parameters,
		CreatedAt:  time.Now(),
	}

	// Set expiration time
	var expiresAt time.Time
	if reportType == string(reports.ReportTypeLowStock) || reportType == string(reports.ReportTypeInventoryValue) {
		expiresAt = time.Now().Add(4 * time.Hour)
	} else {
		expiresAt = time.Now().Add(24 * time.Hour)
	}
	req.ExpiresAt = &expiresAt

	// Generate report synchronously
	result, err := ers.reportManager.GenerateReportSync(ctx, req)
	if err != nil {
		ers.logger.Error("Failed to generate sync report", "error", err, "type", reportType)
		return nil, fmt.Errorf("failed to generate sync report: %w", err)
	}

	ers.logger.Info("Sync report generation completed",
		"request_id", req.ID,
		"result_id", result.ID,
		"type", reportType,
		"processing_time_ms", result.ProcessingTime.Milliseconds())

	return result, nil
}

// GenerateDailySalesReport generates a daily sales report
func (ers *EnhancedReportService) GenerateDailySalesReport(ctx context.Context, date string, async bool) (*reports.ReportResult, error) {
	parameters := make(map[string]interface{})
	if date != "" {
		parameters["date"] = date
	}

	if async {
		return ers.GenerateReportAsync(ctx, string(reports.ReportTypeDailySales), string(reports.ReportFormatJSON), parameters)
	}
	return ers.GenerateReportSync(ctx, string(reports.ReportTypeDailySales), string(reports.ReportFormatJSON), parameters)
}

// GenerateWeeklySalesReport generates a weekly sales report
func (ers *EnhancedReportService) GenerateWeeklySalesReport(ctx context.Context, weekStart string, async bool) (*reports.ReportResult, error) {
	parameters := make(map[string]interface{})
	if weekStart != "" {
		parameters["week_start"] = weekStart
	}

	if async {
		return ers.GenerateReportAsync(ctx, string(reports.ReportTypeWeeklySales), string(reports.ReportFormatJSON), parameters)
	}
	return ers.GenerateReportSync(ctx, string(reports.ReportTypeWeeklySales), string(reports.ReportFormatJSON), parameters)
}

// GenerateMonthlySalesReport generates a monthly sales report
func (ers *EnhancedReportService) GenerateMonthlySalesReport(ctx context.Context, year int, month int, async bool) (*reports.ReportResult, error) {
	parameters := map[string]interface{}{
		"year":  year,
		"month": month,
	}

	if async {
		return ers.GenerateReportAsync(ctx, string(reports.ReportTypeMonthlySales), string(reports.ReportFormatJSON), parameters)
	}
	return ers.GenerateReportSync(ctx, string(reports.ReportTypeMonthlySales), string(reports.ReportFormatJSON), parameters)
}

// GenerateLowStockReport generates a low stock alert report
func (ers *EnhancedReportService) GenerateLowStockReport(ctx context.Context, threshold int, async bool) (*reports.ReportResult, error) {
	parameters := map[string]interface{}{
		"threshold": threshold,
	}

	if async {
		return ers.GenerateReportAsync(ctx, string(reports.ReportTypeLowStock), string(reports.ReportFormatJSON), parameters)
	}
	return ers.GenerateReportSync(ctx, string(reports.ReportTypeLowStock), string(reports.ReportFormatJSON), parameters)
}

// GenerateTopProductsReport generates a top products report
func (ers *EnhancedReportService) GenerateTopProductsReport(ctx context.Context, period string, limit int, async bool) (*reports.ReportResult, error) {
	parameters := map[string]interface{}{
		"period": period,
		"limit":  limit,
	}

	if async {
		return ers.GenerateReportAsync(ctx, string(reports.ReportTypeTopProducts), string(reports.ReportFormatJSON), parameters)
	}
	return ers.GenerateReportSync(ctx, string(reports.ReportTypeTopProducts), string(reports.ReportFormatJSON), parameters)
}

// GenerateCustomerActivityReport generates a customer activity report
func (ers *EnhancedReportService) GenerateCustomerActivityReport(ctx context.Context, period string, async bool) (*reports.ReportResult, error) {
	parameters := map[string]interface{}{
		"period": period,
	}

	if async {
		return ers.GenerateReportAsync(ctx, string(reports.ReportTypeCustomerActivity), string(reports.ReportFormatJSON), parameters)
	}
	return ers.GenerateReportSync(ctx, string(reports.ReportTypeCustomerActivity), string(reports.ReportFormatJSON), parameters)
}

// GenerateInventoryValueReport generates an inventory valuation report
func (ers *EnhancedReportService) GenerateInventoryValueReport(ctx context.Context, async bool) (*reports.ReportResult, error) {
	parameters := make(map[string]interface{})

	if async {
		return ers.GenerateReportAsync(ctx, string(reports.ReportTypeInventoryValue), string(reports.ReportFormatJSON), parameters)
	}
	return ers.GenerateReportSync(ctx, string(reports.ReportTypeInventoryValue), string(reports.ReportFormatJSON), parameters)
}

// GetReportMetrics returns report generation metrics
func (ers *EnhancedReportService) GetReportMetrics() *reports.ReportMetrics {
	return ers.reportManager.GetMetrics()
}

// GetCacheStats returns cache statistics
func (ers *EnhancedReportService) GetCacheStats() map[string]interface{} {
	return ers.reportManager.GetCacheStats()
}

// GetSupportedTypes returns all supported report types
func (ers *EnhancedReportService) GetSupportedTypes() []string {
	supportedTypes := ers.reportManager.GetSupportedTypes()
	types := make([]string, len(supportedTypes))
	for i, reportType := range supportedTypes {
		types[i] = string(reportType)
	}
	return types
}

// FlushCache clears all cached reports
func (ers *EnhancedReportService) FlushCache() {
	ers.reportManager.FlushCache()
	ers.logger.Info("Report cache flushed")
}

// SubmitReportJob submits a report generation job to the worker pool
func (ers *EnhancedReportService) SubmitReportJob(ctx context.Context, reportType string, format string, parameters map[string]interface{}, priority int) (*ReportJobResult, error) {
	// Create report job
	job := &ReportJob{
		BaseJob:    workers.NewBaseJob(workers.JobTypeReportGeneration, priority, 3),
		ReportType: reportType,
		Format:     format,
		Parameters: parameters,
		Service:    ers,
	}

	// Submit to worker pool
	err := ers.poolManager.SubmitJob(job)
	if err != nil {
		ers.logger.Error("Failed to submit report job", "error", err, "type", reportType)
		return nil, fmt.Errorf("failed to submit report job: %w", err)
	}

	result := &ReportJobResult{
		JobID:       job.GetID(),
		ReportType:  reportType,
		Format:      format,
		Status:      "submitted",
		SubmittedAt: time.Now(),
	}

	ers.logger.Info("Report job submitted",
		"job_id", job.GetID(),
		"type", reportType,
		"priority", priority)

	return result, nil
}

// Legacy methods from the base ReportService interface
func (ers *EnhancedReportService) GenerateSalesReport(ctx context.Context, req GenerateSalesReportRequest) (*SalesReportResponse, error) {
	// Convert to new format and generate
	result, err := ers.GenerateDailySalesReport(ctx, "", false) // Default to daily report, sync
	if err != nil {
		return nil, err
	}

	// Convert result to legacy format
	return &SalesReportResponse{
		Report: map[string]interface{}{
			"data":            result.Data,
			"generated_at":    result.GeneratedAt,
			"processing_time": result.ProcessingTime,
		},
	}, nil
}

func (ers *EnhancedReportService) GetSalesReport(ctx context.Context, req GetSalesReportRequest) (*SalesReportResponse, error) {
	// For now, generate a new report (in a real implementation, this might retrieve a stored report)
	return ers.GenerateSalesReport(ctx, GenerateSalesReportRequest{})
}

func (ers *EnhancedReportService) GetLowStockAlert(ctx context.Context, req LowStockRequest) (*LowStockResponse, error) {
	// Generate low stock report
	result, err := ers.GenerateLowStockReport(ctx, req.Threshold, false)
	if err != nil {
		return nil, err
	}

	// Convert to legacy format
	if lowStockData, ok := result.Data.(*reports.LowStockReportData); ok {
		var products []ProductLowStock
		for _, item := range lowStockData.Items {
			products = append(products, ProductLowStock{
				ProductID:    item.ProductID,
				ProductName:  item.ProductName,
				SKU:          item.SKU,
				CurrentStock: item.CurrentStock,
				MinThreshold: item.MinThreshold,
			})
		}

		return &LowStockResponse{
			Products: products,
		}, nil
	}

	return &LowStockResponse{Products: []ProductLowStock{}}, nil
}

// ReportJob represents a report generation job for the worker pool
type ReportJob struct {
	*workers.BaseJob
	ReportType string                 `json:"report_type"`
	Format     string                 `json:"format"`
	Parameters map[string]interface{} `json:"parameters"`
	Service    *EnhancedReportService `json:"-"` // Don't serialize the service
}

// Execute implements the Job interface
func (rj *ReportJob) Execute(ctx context.Context) error {
	_, err := rj.Service.GenerateReportSync(ctx, rj.ReportType, rj.Format, rj.Parameters)
	return err
}

// ReportJobResult represents the result of submitting a report job
type ReportJobResult struct {
	JobID       string    `json:"job_id"`
	ReportType  string    `json:"report_type"`
	Format      string    `json:"format"`
	Status      string    `json:"status"`
	SubmittedAt time.Time `json:"submitted_at"`
}
