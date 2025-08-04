package reports

import (
	"context"
	"fmt"
	"time"

	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/logger"
)

// CustomerReportGenerator generates customer-related reports
type CustomerReportGenerator struct {
	userRepo    repository.UserRepository
	orderRepo   repository.OrderRepository
	paymentRepo repository.PaymentRepository
	logger      *logger.Logger
}

// NewCustomerReportGenerator creates a new customer report generator
func NewCustomerReportGenerator(
	userRepo repository.UserRepository,
	orderRepo repository.OrderRepository,
	paymentRepo repository.PaymentRepository,
	logger *logger.Logger,
) *CustomerReportGenerator {
	return &CustomerReportGenerator{
		userRepo:    userRepo,
		orderRepo:   orderRepo,
		paymentRepo: paymentRepo,
		logger:      logger,
	}
}

// GenerateReport generates a customer report based on the request
func (crg *CustomerReportGenerator) GenerateReport(ctx context.Context, req *ReportRequest) (*ReportResult, error) {
	crg.logger.Info("Generating customer report",
		"type", string(req.Type),
		"id", req.ID)

	startTime := time.Now()

	var reportData interface{}
	var err error

	switch req.Type {
	case ReportTypeCustomerActivity:
		reportData, err = crg.generateCustomerActivityReport(ctx, req.Parameters)
	case ReportTypeOrderAnalytics:
		reportData, err = crg.generateOrderAnalyticsReport(ctx, req.Parameters)
	default:
		return nil, fmt.Errorf("unsupported customer report type: %s", req.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate customer report: %w", err)
	}

	processingTime := time.Since(startTime)

	result := &ReportResult{
		ID:             fmt.Sprintf("customer_%s_%d", req.ID, time.Now().UnixNano()),
		RequestID:      req.ID,
		Type:           req.Type,
		Format:         req.Format,
		Status:         ReportStatusCompleted,
		Data:           reportData,
		ProcessingTime: processingTime,
		RowCount:       crg.getRowCount(reportData),
		Metadata: map[string]interface{}{
			"generator":     "CustomerReportGenerator",
			"processing_ms": processingTime.Milliseconds(),
			"generated_at":  time.Now(),
		},
	}

	crg.logger.Info("Customer report generated successfully",
		"type", string(req.Type),
		"id", req.ID,
		"processing_time_ms", processingTime.Milliseconds(),
		"row_count", result.RowCount)

	return result, nil
}

// GetSupportedTypes returns the report types this generator supports
func (crg *CustomerReportGenerator) GetSupportedTypes() []ReportType {
	return []ReportType{
		ReportTypeCustomerActivity,
		ReportTypeOrderAnalytics,
	}
}

// GetName returns the generator name
func (crg *CustomerReportGenerator) GetName() string {
	return "CustomerReportGenerator"
}

// EstimateGenerationTime estimates how long the report will take to generate
func (crg *CustomerReportGenerator) EstimateGenerationTime(req *ReportRequest) time.Duration {
	switch req.Type {
	case ReportTypeCustomerActivity:
		return 4 * time.Second
	case ReportTypeOrderAnalytics:
		return 6 * time.Second
	default:
		return 5 * time.Second
	}
}

// generateCustomerActivityReport generates a customer activity analytics report
func (crg *CustomerReportGenerator) generateCustomerActivityReport(ctx context.Context, params map[string]interface{}) (*CustomerActivityReportData, error) {
	// Parse parameters
	period := "last_30_days"
	if periodParam, ok := params["period"].(string); ok {
		period = periodParam
	}

	var startDate, endDate time.Time
	now := time.Now()

	switch period {
	case "last_7_days":
		startDate = now.AddDate(0, 0, -7)
		endDate = now
	case "last_30_days":
		startDate = now.AddDate(0, 0, -30)
		endDate = now
	case "last_90_days":
		startDate = now.AddDate(0, 0, -90)
		endDate = now
	case "last_year":
		startDate = now.AddDate(-1, 0, 0)
		endDate = now
	default:
		startDate = now.AddDate(0, 0, -30)
		endDate = now
	}

	crg.logger.Debug("Generating customer activity report",
		"period", period,
		"start_date", startDate,
		"end_date", endDate)

	// Simulate customer analytics data
	totalCustomers := 2847
	activeCustomers := 1923
	newCustomers := 156
	churnRate := 0.084 // 8.4%
	retentionRate := 1.0 - churnRate

	// Generate top customers data
	topCustomers := []CustomerData{
		{
			CustomerID:    "cust-001",
			CustomerName:  "John Smith",
			Email:         "john.smith@example.com",
			OrderCount:    23,
			TotalSpent:    12450.75,
			AvgOrderValue: 541.34,
			LastOrderDate: now.AddDate(0, 0, -2),
			Rank:          1,
			Segment:       "premium",
		},
		{
			CustomerID:    "cust-002",
			CustomerName:  "Sarah Johnson",
			Email:         "sarah.j@example.com",
			OrderCount:    18,
			TotalSpent:    9876.50,
			AvgOrderValue: 548.69,
			LastOrderDate: now.AddDate(0, 0, -1),
			Rank:          2,
			Segment:       "premium",
		},
		{
			CustomerID:    "cust-003",
			CustomerName:  "Michael Brown",
			Email:         "m.brown@example.com",
			OrderCount:    31,
			TotalSpent:    8942.25,
			AvgOrderValue: 288.46,
			LastOrderDate: now.AddDate(0, 0, -3),
			Rank:          3,
			Segment:       "regular",
		},
		{
			CustomerID:    "cust-004",
			CustomerName:  "Emily Davis",
			Email:         "emily.davis@example.com",
			OrderCount:    15,
			TotalSpent:    7823.40,
			AvgOrderValue: 521.56,
			LastOrderDate: now.AddDate(0, 0, -5),
			Rank:          4,
			Segment:       "premium",
		},
		{
			CustomerID:    "cust-005",
			CustomerName:  "David Wilson",
			Email:         "d.wilson@example.com",
			OrderCount:    27,
			TotalSpent:    6754.30,
			AvgOrderValue: 250.16,
			LastOrderDate: now.AddDate(0, 0, -1),
			Rank:          5,
			Segment:       "regular",
		},
		{
			CustomerID:    "cust-006",
			CustomerName:  "Lisa Anderson",
			Email:         "lisa.a@example.com",
			OrderCount:    12,
			TotalSpent:    6543.20,
			AvgOrderValue: 545.27,
			LastOrderDate: now.AddDate(0, 0, -4),
			Rank:          6,
			Segment:       "premium",
		},
		{
			CustomerID:    "cust-007",
			CustomerName:  "Robert Taylor",
			Email:         "r.taylor@example.com",
			OrderCount:    22,
			TotalSpent:    5876.45,
			AvgOrderValue: 267.11,
			LastOrderDate: now.AddDate(0, 0, -6),
			Rank:          7,
			Segment:       "regular",
		},
		{
			CustomerID:    "cust-008",
			CustomerName:  "Jennifer Moore",
			Email:         "jen.moore@example.com",
			OrderCount:    8,
			TotalSpent:    5432.10,
			AvgOrderValue: 679.01,
			LastOrderDate: now.AddDate(0, 0, -2),
			Rank:          8,
			Segment:       "premium",
		},
		{
			CustomerID:    "cust-009",
			CustomerName:  "Christopher Lee",
			Email:         "c.lee@example.com",
			OrderCount:    19,
			TotalSpent:    4987.65,
			AvgOrderValue: 262.51,
			LastOrderDate: now.AddDate(0, 0, -8),
			Rank:          9,
			Segment:       "regular",
		},
		{
			CustomerID:    "cust-010",
			CustomerName:  "Amanda White",
			Email:         "amanda.w@example.com",
			OrderCount:    11,
			TotalSpent:    4756.80,
			AvgOrderValue: 432.44,
			LastOrderDate: now.AddDate(0, 0, -3),
			Rank:          10,
			Segment:       "regular",
		},
	}

	// Generate daily activity data
	var activityByDay []DailyActivityData
	days := int(endDate.Sub(startDate).Hours() / 24)

	for i := 0; i < days; i++ {
		day := startDate.AddDate(0, 0, i)

		// Simulate varying daily activity
		dayOfWeek := day.Weekday()
		baseActivity := 85
		if dayOfWeek == time.Saturday || dayOfWeek == time.Sunday {
			baseActivity = 45 // Lower weekend activity
		}

		variance := (i % 7) * 5 // Weekly pattern
		dailyActive := baseActivity + variance
		dailyNew := int(float64(dailyActive) * 0.08)   // 8% new customers
		dailyOrders := int(float64(dailyActive) * 1.3) // 1.3 orders per active customer
		dailyRevenue := float64(dailyOrders) * 275.50  // Average order value

		activityByDay = append(activityByDay, DailyActivityData{
			Date:            day,
			ActiveCustomers: dailyActive,
			NewCustomers:    dailyNew,
			OrderCount:      dailyOrders,
			Revenue:         dailyRevenue,
		})
	}

	report := &CustomerActivityReportData{
		Period:          fmt.Sprintf("Customer Activity - %s", period),
		StartDate:       startDate,
		EndDate:         endDate,
		TotalCustomers:  totalCustomers,
		ActiveCustomers: activeCustomers,
		NewCustomers:    newCustomers,
		ChurnRate:       churnRate,
		RetentionRate:   retentionRate,
		TopCustomers:    topCustomers,
		ActivityByDay:   activityByDay,
		Summary: map[string]interface{}{
			"customer_growth_rate":      0.067, // 6.7%
			"avg_customer_lifetime":     "14.2 months",
			"customer_acquisition_cost": "$45.30",
			"customer_lifetime_value":   "$2,847.50",
			"most_active_day":           "Tuesday",
			"premium_customers_percent": 35.4,
			"repeat_purchase_rate":      71.2,
			"avg_days_between_orders":   18.5,
		},
	}

	return report, nil
}

// generateOrderAnalyticsReport generates an order analytics report
func (crg *CustomerReportGenerator) generateOrderAnalyticsReport(ctx context.Context, params map[string]interface{}) (*OrderAnalyticsReportData, error) {
	// Parse parameters
	period := "last_30_days"
	if periodParam, ok := params["period"].(string); ok {
		period = periodParam
	}

	var startDate, endDate time.Time
	now := time.Now()

	switch period {
	case "last_7_days":
		startDate = now.AddDate(0, 0, -7)
		endDate = now
	case "last_30_days":
		startDate = now.AddDate(0, 0, -30)
		endDate = now
	case "last_90_days":
		startDate = now.AddDate(0, 0, -90)
		endDate = now
	default:
		startDate = now.AddDate(0, 0, -30)
		endDate = now
	}

	crg.logger.Debug("Generating order analytics report",
		"period", period,
		"start_date", startDate,
		"end_date", endDate)

	// Simulate order analytics data
	totalOrders := 1456
	completedOrders := 1398
	cancelledOrders := 47
	pendingOrders := 11
	avgOrderValue := 287.45
	totalRevenue := float64(completedOrders) * avgOrderValue

	// Order status breakdown
	orderStatusBreakdown := []OrderStatusData{
		{
			Status:     "completed",
			Count:      completedOrders,
			Percentage: (float64(completedOrders) / float64(totalOrders)) * 100,
			Revenue:    totalRevenue,
		},
		{
			Status:     "pending",
			Count:      pendingOrders,
			Percentage: (float64(pendingOrders) / float64(totalOrders)) * 100,
			Revenue:    float64(pendingOrders) * avgOrderValue,
		},
		{
			Status:     "cancelled",
			Count:      cancelledOrders,
			Percentage: (float64(cancelledOrders) / float64(totalOrders)) * 100,
			Revenue:    0,
		},
	}

	// Order size distribution
	orderSizeDistribution := []OrderSizeData{
		{
			SizeRange:    "$0 - $100",
			Count:        423,
			Percentage:   29.1,
			AvgValue:     67.85,
			TotalRevenue: 28700.55,
		},
		{
			SizeRange:    "$101 - $250",
			Count:        511,
			Percentage:   35.1,
			AvgValue:     175.32,
			TotalRevenue: 89589.52,
		},
		{
			SizeRange:    "$251 - $500",
			Count:        378,
			Percentage:   26.0,
			AvgValue:     367.89,
			TotalRevenue: 139062.42,
		},
		{
			SizeRange:    "$501 - $1000",
			Count:        108,
			Percentage:   7.4,
			AvgValue:     742.15,
			TotalRevenue: 80152.20,
		},
		{
			SizeRange:    "$1000+",
			Count:        36,
			Percentage:   2.5,
			AvgValue:     1456.78,
			TotalRevenue: 52444.08,
		},
	}

	// Hourly order patterns
	hourlyPatterns := []HourlyOrderData{
		{Hour: 0, OrderCount: 12, Revenue: 3449.40},
		{Hour: 1, OrderCount: 8, Revenue: 2299.60},
		{Hour: 2, OrderCount: 5, Revenue: 1437.25},
		{Hour: 3, OrderCount: 3, Revenue: 862.35},
		{Hour: 4, OrderCount: 4, Revenue: 1149.80},
		{Hour: 5, OrderCount: 7, Revenue: 2012.15},
		{Hour: 6, OrderCount: 15, Revenue: 4311.75},
		{Hour: 7, OrderCount: 28, Revenue: 8048.60},
		{Hour: 8, OrderCount: 45, Revenue: 12935.25},
		{Hour: 9, OrderCount: 67, Revenue: 19259.15},
		{Hour: 10, OrderCount: 89, Revenue: 25583.05},
		{Hour: 11, OrderCount: 94, Revenue: 27020.30},
		{Hour: 12, OrderCount: 102, Revenue: 29319.90},
		{Hour: 13, OrderCount: 96, Revenue: 27595.20},
		{Hour: 14, OrderCount: 78, Revenue: 22421.10},
		{Hour: 15, OrderCount: 85, Revenue: 24433.25},
		{Hour: 16, OrderCount: 91, Revenue: 26157.95},
		{Hour: 17, OrderCount: 88, Revenue: 25295.60},
		{Hour: 18, OrderCount: 82, Revenue: 23570.90},
		{Hour: 19, OrderCount: 74, Revenue: 21271.30},
		{Hour: 20, OrderCount: 65, Revenue: 18684.25},
		{Hour: 21, OrderCount: 56, Revenue: 16096.20},
		{Hour: 22, OrderCount: 38, Revenue: 10923.10},
		{Hour: 23, OrderCount: 23, Revenue: 6611.35},
	}

	report := &OrderAnalyticsReportData{
		Period:                fmt.Sprintf("Order Analytics - %s", period),
		StartDate:             startDate,
		EndDate:               endDate,
		TotalOrders:           totalOrders,
		CompletedOrders:       completedOrders,
		CancelledOrders:       cancelledOrders,
		PendingOrders:         pendingOrders,
		AverageOrderValue:     avgOrderValue,
		TotalRevenue:          totalRevenue,
		OrderStatusBreakdown:  orderStatusBreakdown,
		OrderSizeDistribution: orderSizeDistribution,
		HourlyPatterns:        hourlyPatterns,
		Summary: map[string]interface{}{
			"completion_rate":       95.95, // %
			"cancellation_rate":     3.23,  // %
			"peak_hour":             "12:00 PM",
			"avg_processing_time":   "2.3 hours",
			"same_day_shipping":     78.5, // %
			"mobile_orders_percent": 45.2,
			"repeat_orders_percent": 67.8,
			"international_orders":  8.3,
		},
	}

	return report, nil
}

// getRowCount estimates the number of rows in the report data
func (crg *CustomerReportGenerator) getRowCount(data interface{}) int {
	switch reportData := data.(type) {
	case *CustomerActivityReportData:
		return len(reportData.TopCustomers) + len(reportData.ActivityByDay)
	case *OrderAnalyticsReportData:
		return len(reportData.OrderStatusBreakdown) + len(reportData.OrderSizeDistribution) + len(reportData.HourlyPatterns)
	default:
		return 0
	}
}

// OrderAnalyticsReportData represents order analytics report data
type OrderAnalyticsReportData struct {
	Period                string                 `json:"period"`
	StartDate             time.Time              `json:"start_date"`
	EndDate               time.Time              `json:"end_date"`
	TotalOrders           int                    `json:"total_orders"`
	CompletedOrders       int                    `json:"completed_orders"`
	CancelledOrders       int                    `json:"cancelled_orders"`
	PendingOrders         int                    `json:"pending_orders"`
	AverageOrderValue     float64                `json:"average_order_value"`
	TotalRevenue          float64                `json:"total_revenue"`
	OrderStatusBreakdown  []OrderStatusData      `json:"order_status_breakdown"`
	OrderSizeDistribution []OrderSizeData        `json:"order_size_distribution"`
	HourlyPatterns        []HourlyOrderData      `json:"hourly_patterns"`
	Summary               map[string]interface{} `json:"summary"`
}

// OrderStatusData represents order status breakdown
type OrderStatusData struct {
	Status     string  `json:"status"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
	Revenue    float64 `json:"revenue"`
}

// OrderSizeData represents order size distribution
type OrderSizeData struct {
	SizeRange    string  `json:"size_range"`
	Count        int     `json:"count"`
	Percentage   float64 `json:"percentage"`
	AvgValue     float64 `json:"avg_value"`
	TotalRevenue float64 `json:"total_revenue"`
}

// HourlyOrderData represents hourly order patterns
type HourlyOrderData struct {
	Hour       int     `json:"hour"`
	OrderCount int     `json:"order_count"`
	Revenue    float64 `json:"revenue"`
}
