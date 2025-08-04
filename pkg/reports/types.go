package reports

import (
	"time"
)

// ReportType defines different types of reports that can be generated
type ReportType string

const (
	ReportTypeDailySales       ReportType = "daily_sales"
	ReportTypeWeeklySales      ReportType = "weekly_sales"
	ReportTypeMonthlySales     ReportType = "monthly_sales"
	ReportTypeLowStock         ReportType = "low_stock"
	ReportTypeTopProducts      ReportType = "top_products"
	ReportTypeCustomerActivity ReportType = "customer_activity"
	ReportTypeInventoryValue   ReportType = "inventory_value"
	ReportTypeOrderAnalytics   ReportType = "order_analytics"
	ReportTypePaymentAnalytics ReportType = "payment_analytics"
	ReportTypeRevenue          ReportType = "revenue"
)

// ReportStatus represents the status of report generation
type ReportStatus string

const (
	ReportStatusPending    ReportStatus = "pending"
	ReportStatusGenerating ReportStatus = "generating"
	ReportStatusCompleted  ReportStatus = "completed"
	ReportStatusFailed     ReportStatus = "failed"
	ReportStatusCancelled  ReportStatus = "cancelled"
	ReportStatusExpired    ReportStatus = "expired"
)

// ReportFormat defines the output format for reports
type ReportFormat string

const (
	ReportFormatJSON ReportFormat = "json"
	ReportFormatCSV  ReportFormat = "csv"
	ReportFormatPDF  ReportFormat = "pdf"
	ReportFormatXLSX ReportFormat = "xlsx"
)

// ReportPriority defines the priority level for report generation
type ReportPriority int

const (
	ReportPriorityLow      ReportPriority = 1
	ReportPriorityNormal   ReportPriority = 5
	ReportPriorityHigh     ReportPriority = 8
	ReportPriorityCritical ReportPriority = 10
)

// ReportRequest represents a request to generate a report
type ReportRequest struct {
	ID          string                 `json:"id"`
	Type        ReportType             `json:"type"`
	Format      ReportFormat           `json:"format"`
	Priority    ReportPriority         `json:"priority"`
	Parameters  map[string]interface{} `json:"parameters"`
	UserID      string                 `json:"user_id,omitempty"`
	Email       string                 `json:"email,omitempty"`
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ReportResult represents the result of report generation
type ReportResult struct {
	ID             string                 `json:"id"`
	RequestID      string                 `json:"request_id"`
	Type           ReportType             `json:"type"`
	Format         ReportFormat           `json:"format"`
	Status         ReportStatus           `json:"status"`
	Data           interface{}            `json:"data,omitempty"`
	FilePath       string                 `json:"file_path,omitempty"`
	FileSize       int64                  `json:"file_size,omitempty"`
	RowCount       int                    `json:"row_count,omitempty"`
	GeneratedAt    *time.Time             `json:"generated_at,omitempty"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Error          string                 `json:"error,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ReportMetrics tracks report generation performance
type ReportMetrics struct {
	TotalReports     int64         `json:"total_reports"`
	CompletedReports int64         `json:"completed_reports"`
	FailedReports    int64         `json:"failed_reports"`
	PendingReports   int64         `json:"pending_reports"`
	AverageGenTime   time.Duration `json:"average_generation_time"`
	TotalGenTime     time.Duration `json:"total_generation_time"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	ActiveGenerators int           `json:"active_generators"`
	QueueDepth       int           `json:"queue_depth"`
}

// SalesReportData represents daily/weekly/monthly sales report data
type SalesReportData struct {
	Period            string                 `json:"period"`
	StartDate         time.Time              `json:"start_date"`
	EndDate           time.Time              `json:"end_date"`
	TotalRevenue      float64                `json:"total_revenue"`
	TotalOrders       int                    `json:"total_orders"`
	AverageOrderValue float64                `json:"average_order_value"`
	TopProducts       []ProductSalesData     `json:"top_products"`
	SalesByDay        []DailySalesData       `json:"sales_by_day"`
	PaymentMethods    []PaymentMethodData    `json:"payment_methods"`
	CustomerSegments  []CustomerSegmentData  `json:"customer_segments"`
	Summary           map[string]interface{} `json:"summary"`
}

// ProductSalesData represents product sales information
type ProductSalesData struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	SKU         string  `json:"sku"`
	Quantity    int     `json:"quantity"`
	Revenue     float64 `json:"revenue"`
	OrderCount  int     `json:"order_count"`
	AvgPrice    float64 `json:"avg_price"`
	Rank        int     `json:"rank"`
}

// DailySalesData represents daily sales breakdown
type DailySalesData struct {
	Date          time.Time `json:"date"`
	Revenue       float64   `json:"revenue"`
	OrderCount    int       `json:"order_count"`
	CustomerCount int       `json:"customer_count"`
	AvgOrderValue float64   `json:"avg_order_value"`
}

// PaymentMethodData represents payment method analytics
type PaymentMethodData struct {
	Method        string  `json:"method"`
	OrderCount    int     `json:"order_count"`
	Revenue       float64 `json:"revenue"`
	Percentage    float64 `json:"percentage"`
	AvgOrderValue float64 `json:"avg_order_value"`
}

// CustomerSegmentData represents customer segment analytics
type CustomerSegmentData struct {
	Segment       string  `json:"segment"`
	CustomerCount int     `json:"customer_count"`
	Revenue       float64 `json:"revenue"`
	OrderCount    int     `json:"order_count"`
	AvgOrderValue float64 `json:"avg_order_value"`
	Percentage    float64 `json:"percentage"`
}

// LowStockReportData represents low stock alert report data
type LowStockReportData struct {
	GeneratedAt   time.Time              `json:"generated_at"`
	Threshold     int                    `json:"threshold"`
	TotalItems    int                    `json:"total_items"`
	CriticalItems int                    `json:"critical_items"`
	WarningItems  int                    `json:"warning_items"`
	Items         []LowStockItemData     `json:"items"`
	Summary       map[string]interface{} `json:"summary"`
}

// LowStockItemData represents individual low stock item information
type LowStockItemData struct {
	ProductID        string     `json:"product_id"`
	ProductName      string     `json:"product_name"`
	SKU              string     `json:"sku"`
	CurrentStock     int        `json:"current_stock"`
	ReservedStock    int        `json:"reserved_stock"`
	AvailableStock   int        `json:"available_stock"`
	MinThreshold     int        `json:"min_threshold"`
	StockLevel       string     `json:"stock_level"` // "critical", "warning", "ok"
	LastRestocked    *time.Time `json:"last_restocked,omitempty"`
	DaysOutOfStock   int        `json:"days_out_of_stock"`
	EstimatedDemand  int        `json:"estimated_demand"`
	RecommendedOrder int        `json:"recommended_order"`
}

// TopProductsReportData represents top-selling products report
type TopProductsReportData struct {
	Period      string                 `json:"period"`
	StartDate   time.Time              `json:"start_date"`
	EndDate     time.Time              `json:"end_date"`
	TopProducts []ProductSalesData     `json:"top_products"`
	Categories  []CategoryData         `json:"categories"`
	Summary     map[string]interface{} `json:"summary"`
}

// CategoryData represents category performance data
type CategoryData struct {
	CategoryName string  `json:"category_name"`
	ProductCount int     `json:"product_count"`
	Revenue      float64 `json:"revenue"`
	OrderCount   int     `json:"order_count"`
	AvgPrice     float64 `json:"avg_price"`
	Percentage   float64 `json:"percentage"`
}

// CustomerActivityReportData represents customer activity analytics
type CustomerActivityReportData struct {
	Period          string                 `json:"period"`
	StartDate       time.Time              `json:"start_date"`
	EndDate         time.Time              `json:"end_date"`
	TotalCustomers  int                    `json:"total_customers"`
	ActiveCustomers int                    `json:"active_customers"`
	NewCustomers    int                    `json:"new_customers"`
	ChurnRate       float64                `json:"churn_rate"`
	RetentionRate   float64                `json:"retention_rate"`
	TopCustomers    []CustomerData         `json:"top_customers"`
	ActivityByDay   []DailyActivityData    `json:"activity_by_day"`
	Summary         map[string]interface{} `json:"summary"`
}

// CustomerData represents individual customer analytics
type CustomerData struct {
	CustomerID    string    `json:"customer_id"`
	CustomerName  string    `json:"customer_name"`
	Email         string    `json:"email"`
	OrderCount    int       `json:"order_count"`
	TotalSpent    float64   `json:"total_spent"`
	AvgOrderValue float64   `json:"avg_order_value"`
	LastOrderDate time.Time `json:"last_order_date"`
	Rank          int       `json:"rank"`
	Segment       string    `json:"segment"`
}

// DailyActivityData represents daily customer activity
type DailyActivityData struct {
	Date            time.Time `json:"date"`
	ActiveCustomers int       `json:"active_customers"`
	NewCustomers    int       `json:"new_customers"`
	OrderCount      int       `json:"order_count"`
	Revenue         float64   `json:"revenue"`
}

// ReportCache represents cached report data
type ReportCache struct {
	Key          string                 `json:"key"`
	ReportType   ReportType             `json:"report_type"`
	Parameters   map[string]interface{} `json:"parameters"`
	Data         interface{}            `json:"data"`
	GeneratedAt  time.Time              `json:"generated_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	HitCount     int                    `json:"hit_count"`
	LastAccessed time.Time              `json:"last_accessed"`
}

// IsExpired checks if the cached report has expired
func (rc *ReportCache) IsExpired() bool {
	return time.Now().After(rc.ExpiresAt)
}

// ShouldRefresh determines if the cache should be refreshed based on age and access patterns
func (rc *ReportCache) ShouldRefresh() bool {
	age := time.Since(rc.GeneratedAt)

	// Refresh if expired
	if rc.IsExpired() {
		return true
	}

	// Refresh if older than 4 hours and frequently accessed
	if age > 4*time.Hour && rc.HitCount > 10 {
		return true
	}

	// Refresh if older than 8 hours regardless of access
	if age > 8*time.Hour {
		return true
	}

	return false
}

// ReportSchedule represents a scheduled report generation
type ReportSchedule struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	ReportType ReportType             `json:"report_type"`
	Format     ReportFormat           `json:"format"`
	Parameters map[string]interface{} `json:"parameters"`
	Schedule   string                 `json:"schedule"` // Cron expression
	Recipients []string               `json:"recipients"`
	IsActive   bool                   `json:"is_active"`
	LastRun    *time.Time             `json:"last_run,omitempty"`
	NextRun    *time.Time             `json:"next_run,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}
