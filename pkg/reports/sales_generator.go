package reports

import (
	"context"
	"fmt"
	"time"

	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/logger"
)

// SalesReportGenerator generates sales-related reports
type SalesReportGenerator struct {
	orderRepo   repository.OrderRepository
	paymentRepo repository.PaymentRepository
	userRepo    repository.UserRepository
	productRepo repository.ProductRepository
	logger      *logger.Logger
}

// NewSalesReportGenerator creates a new sales report generator
func NewSalesReportGenerator(
	orderRepo repository.OrderRepository,
	paymentRepo repository.PaymentRepository,
	userRepo repository.UserRepository,
	productRepo repository.ProductRepository,
	logger *logger.Logger,
) *SalesReportGenerator {
	return &SalesReportGenerator{
		orderRepo:   orderRepo,
		paymentRepo: paymentRepo,
		userRepo:    userRepo,
		productRepo: productRepo,
		logger:      logger,
	}
}

// GenerateReport generates a sales report based on the request
func (srg *SalesReportGenerator) GenerateReport(ctx context.Context, req *ReportRequest) (*ReportResult, error) {
	srg.logger.Info("Generating sales report",
		"type", string(req.Type),
		"id", req.ID)

	startTime := time.Now()

	var reportData interface{}
	var err error

	switch req.Type {
	case ReportTypeDailySales:
		reportData, err = srg.generateDailySalesReport(ctx, req.Parameters)
	case ReportTypeWeeklySales:
		reportData, err = srg.generateWeeklySalesReport(ctx, req.Parameters)
	case ReportTypeMonthlySales:
		reportData, err = srg.generateMonthlySalesReport(ctx, req.Parameters)
	case ReportTypeTopProducts:
		reportData, err = srg.generateTopProductsReport(ctx, req.Parameters)
	case ReportTypeRevenue:
		reportData, err = srg.generateRevenueReport(ctx, req.Parameters)
	default:
		return nil, fmt.Errorf("unsupported sales report type: %s", req.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate sales report: %w", err)
	}

	processingTime := time.Since(startTime)

	result := &ReportResult{
		ID:             fmt.Sprintf("sales_%s_%d", req.ID, time.Now().UnixNano()),
		RequestID:      req.ID,
		Type:           req.Type,
		Format:         req.Format,
		Status:         ReportStatusCompleted,
		Data:           reportData,
		ProcessingTime: processingTime,
		RowCount:       srg.getRowCount(reportData),
		Metadata: map[string]interface{}{
			"generator":     "SalesReportGenerator",
			"processing_ms": processingTime.Milliseconds(),
			"generated_at":  time.Now(),
		},
	}

	srg.logger.Info("Sales report generated successfully",
		"type", string(req.Type),
		"id", req.ID,
		"processing_time_ms", processingTime.Milliseconds(),
		"row_count", result.RowCount)

	return result, nil
}

// GetSupportedTypes returns the report types this generator supports
func (srg *SalesReportGenerator) GetSupportedTypes() []ReportType {
	return []ReportType{
		ReportTypeDailySales,
		ReportTypeWeeklySales,
		ReportTypeMonthlySales,
		ReportTypeTopProducts,
		ReportTypeRevenue,
	}
}

// GetName returns the generator name
func (srg *SalesReportGenerator) GetName() string {
	return "SalesReportGenerator"
}

// EstimateGenerationTime estimates how long the report will take to generate
func (srg *SalesReportGenerator) EstimateGenerationTime(req *ReportRequest) time.Duration {
	switch req.Type {
	case ReportTypeDailySales:
		return 2 * time.Second
	case ReportTypeWeeklySales:
		return 5 * time.Second
	case ReportTypeMonthlySales:
		return 15 * time.Second
	case ReportTypeTopProducts:
		return 3 * time.Second
	case ReportTypeRevenue:
		return 8 * time.Second
	default:
		return 5 * time.Second
	}
}

// generateDailySalesReport generates a daily sales report
func (srg *SalesReportGenerator) generateDailySalesReport(ctx context.Context, params map[string]interface{}) (*SalesReportData, error) {
	// Parse parameters
	date := time.Now().Truncate(24 * time.Hour)
	if dateStr, ok := params["date"].(string); ok {
		if parsedDate, err := time.Parse("2006-01-02", dateStr); err == nil {
			date = parsedDate
		}
	}

	startDate := date
	endDate := date.Add(24 * time.Hour)

	srg.logger.Debug("Generating daily sales report",
		"start_date", startDate,
		"end_date", endDate)

	// Simulate data aggregation (in real implementation, this would query the database)
	totalRevenue := 15750.50
	totalOrders := 45
	avgOrderValue := totalRevenue / float64(totalOrders)

	// Generate mock data for demonstration
	topProducts := []ProductSalesData{
		{
			ProductID:   "prod-001",
			ProductName: "Premium Widget",
			SKU:         "PWD-001",
			Quantity:    25,
			Revenue:     5500.00,
			OrderCount:  15,
			AvgPrice:    220.00,
			Rank:        1,
		},
		{
			ProductID:   "prod-002",
			ProductName: "Standard Widget",
			SKU:         "SWD-002",
			Quantity:    42,
			Revenue:     4200.00,
			OrderCount:  18,
			AvgPrice:    100.00,
			Rank:        2,
		},
		{
			ProductID:   "prod-003",
			ProductName: "Economy Widget",
			SKU:         "EWD-003",
			Quantity:    68,
			Revenue:     3400.00,
			OrderCount:  20,
			AvgPrice:    50.00,
			Rank:        3,
		},
	}

	salesByDay := []DailySalesData{
		{
			Date:          date,
			Revenue:       totalRevenue,
			OrderCount:    totalOrders,
			CustomerCount: 38,
			AvgOrderValue: avgOrderValue,
		},
	}

	paymentMethods := []PaymentMethodData{
		{
			Method:        "credit_card",
			OrderCount:    30,
			Revenue:       10500.35,
			Percentage:    66.7,
			AvgOrderValue: 350.01,
		},
		{
			Method:        "paypal",
			OrderCount:    10,
			Revenue:       3250.15,
			Percentage:    20.6,
			AvgOrderValue: 325.02,
		},
		{
			Method:        "bank_transfer",
			OrderCount:    5,
			Revenue:       2000.00,
			Percentage:    12.7,
			AvgOrderValue: 400.00,
		},
	}

	customerSegments := []CustomerSegmentData{
		{
			Segment:       "premium",
			CustomerCount: 8,
			Revenue:       7875.25,
			OrderCount:    12,
			AvgOrderValue: 656.27,
			Percentage:    50.0,
		},
		{
			Segment:       "regular",
			CustomerCount: 20,
			Revenue:       5512.50,
			OrderCount:    22,
			AvgOrderValue: 250.57,
			Percentage:    35.0,
		},
		{
			Segment:       "new",
			CustomerCount: 10,
			Revenue:       2362.75,
			OrderCount:    11,
			AvgOrderValue: 214.80,
			Percentage:    15.0,
		},
	}

	report := &SalesReportData{
		Period:            fmt.Sprintf("Daily - %s", date.Format("2006-01-02")),
		StartDate:         startDate,
		EndDate:           endDate,
		TotalRevenue:      totalRevenue,
		TotalOrders:       totalOrders,
		AverageOrderValue: avgOrderValue,
		TopProducts:       topProducts,
		SalesByDay:        salesByDay,
		PaymentMethods:    paymentMethods,
		CustomerSegments:  customerSegments,
		Summary: map[string]interface{}{
			"best_selling_product": topProducts[0].ProductName,
			"conversion_rate":      0.084, // 8.4%
			"repeat_customer_rate": 0.632, // 63.2%
			"growth_rate":          0.125, // 12.5% compared to previous day
		},
	}

	return report, nil
}

// generateWeeklySalesReport generates a weekly sales report
func (srg *SalesReportGenerator) generateWeeklySalesReport(ctx context.Context, params map[string]interface{}) (*SalesReportData, error) {
	// Parse week start date
	startDate := time.Now().AddDate(0, 0, -7).Truncate(24 * time.Hour)
	if weekStr, ok := params["week_start"].(string); ok {
		if parsedDate, err := time.Parse("2006-01-02", weekStr); err == nil {
			startDate = parsedDate
		}
	}

	endDate := startDate.AddDate(0, 0, 7)

	srg.logger.Debug("Generating weekly sales report",
		"start_date", startDate,
		"end_date", endDate)

	// Simulate weekly aggregation
	totalRevenue := 87425.75
	totalOrders := 312
	avgOrderValue := totalRevenue / float64(totalOrders)

	// Generate daily breakdown
	var salesByDay []DailySalesData
	for i := 0; i < 7; i++ {
		day := startDate.AddDate(0, 0, i)
		dailyRevenue := totalRevenue * (0.1 + float64(i%3)*0.05) // Varying daily sales
		dailyOrders := int(float64(totalOrders) * (0.1 + float64(i%3)*0.05))
		dailyCustomers := int(float64(dailyOrders) * 0.85) // Some customers place multiple orders

		salesByDay = append(salesByDay, DailySalesData{
			Date:          day,
			Revenue:       dailyRevenue,
			OrderCount:    dailyOrders,
			CustomerCount: dailyCustomers,
			AvgOrderValue: dailyRevenue / float64(dailyOrders),
		})
	}

	topProducts := []ProductSalesData{
		{
			ProductID:   "prod-001",
			ProductName: "Premium Widget",
			SKU:         "PWD-001",
			Quantity:    156,
			Revenue:     34320.00,
			OrderCount:  78,
			AvgPrice:    220.00,
			Rank:        1,
		},
		{
			ProductID:   "prod-002",
			ProductName: "Standard Widget",
			SKU:         "SWD-002",
			Quantity:    234,
			Revenue:     23400.00,
			OrderCount:  89,
			AvgPrice:    100.00,
			Rank:        2,
		},
		{
			ProductID:   "prod-003",
			ProductName: "Economy Widget",
			SKU:         "EWD-003",
			Quantity:    378,
			Revenue:     18900.00,
			OrderCount:  98,
			AvgPrice:    50.00,
			Rank:        3,
		},
	}

	paymentMethods := []PaymentMethodData{
		{
			Method:        "credit_card",
			OrderCount:    210,
			Revenue:       58197.52,
			Percentage:    66.6,
			AvgOrderValue: 277.13,
		},
		{
			Method:        "paypal",
			OrderCount:    65,
			Revenue:       18706.44,
			Percentage:    21.4,
			AvgOrderValue: 287.79,
		},
		{
			Method:        "bank_transfer",
			OrderCount:    37,
			Revenue:       10521.79,
			Percentage:    12.0,
			AvgOrderValue: 284.37,
		},
	}

	customerSegments := []CustomerSegmentData{
		{
			Segment:       "premium",
			CustomerCount: 45,
			Revenue:       43712.88,
			OrderCount:    67,
			AvgOrderValue: 652.43,
			Percentage:    50.0,
		},
		{
			Segment:       "regular",
			CustomerCount: 125,
			Revenue:       30597.01,
			OrderCount:    156,
			AvgOrderValue: 196.14,
			Percentage:    35.0,
		},
		{
			Segment:       "new",
			CustomerCount: 68,
			Revenue:       13115.86,
			OrderCount:    89,
			AvgOrderValue: 147.37,
			Percentage:    15.0,
		},
	}

	report := &SalesReportData{
		Period:            fmt.Sprintf("Weekly - %s to %s", startDate.Format("2006-01-02"), endDate.AddDate(0, 0, -1).Format("2006-01-02")),
		StartDate:         startDate,
		EndDate:           endDate,
		TotalRevenue:      totalRevenue,
		TotalOrders:       totalOrders,
		AverageOrderValue: avgOrderValue,
		TopProducts:       topProducts,
		SalesByDay:        salesByDay,
		PaymentMethods:    paymentMethods,
		CustomerSegments:  customerSegments,
		Summary: map[string]interface{}{
			"best_day":             salesByDay[0].Date.Format("Monday"),
			"conversion_rate":      0.092, // 9.2%
			"repeat_customer_rate": 0.714, // 71.4%
			"growth_rate":          0.086, // 8.6% compared to previous week
		},
	}

	return report, nil
}

// generateMonthlySalesReport generates a monthly sales report
func (srg *SalesReportGenerator) generateMonthlySalesReport(ctx context.Context, params map[string]interface{}) (*SalesReportData, error) {
	// Parse month
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if yearParam, ok := params["year"].(float64); ok {
		year = int(yearParam)
	}
	if monthParam, ok := params["month"].(float64); ok {
		month = int(monthParam)
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	srg.logger.Debug("Generating monthly sales report",
		"year", year,
		"month", month,
		"start_date", startDate,
		"end_date", endDate)

	// Simulate monthly aggregation
	totalRevenue := 367890.25
	totalOrders := 1456
	avgOrderValue := totalRevenue / float64(totalOrders)

	// Generate daily breakdown for the month
	var salesByDay []DailySalesData
	daysInMonth := endDate.Sub(startDate).Hours() / 24
	for i := 0; i < int(daysInMonth); i++ {
		day := startDate.AddDate(0, 0, i)
		// Simulate varying daily sales with weekend patterns
		dayOfWeek := day.Weekday()
		multiplier := 1.0
		if dayOfWeek == time.Saturday || dayOfWeek == time.Sunday {
			multiplier = 0.7 // Lower weekend sales
		}

		dailyRevenue := (totalRevenue / daysInMonth) * multiplier
		dailyOrders := int((float64(totalOrders) / daysInMonth) * multiplier)
		dailyCustomers := int(float64(dailyOrders) * 0.82)

		salesByDay = append(salesByDay, DailySalesData{
			Date:          day,
			Revenue:       dailyRevenue,
			OrderCount:    dailyOrders,
			CustomerCount: dailyCustomers,
			AvgOrderValue: dailyRevenue / float64(dailyOrders),
		})
	}

	topProducts := []ProductSalesData{
		{
			ProductID:   "prod-001",
			ProductName: "Premium Widget",
			SKU:         "PWD-001",
			Quantity:    685,
			Revenue:     150700.00,
			OrderCount:  342,
			AvgPrice:    220.00,
			Rank:        1,
		},
		{
			ProductID:   "prod-002",
			ProductName: "Standard Widget",
			SKU:         "SWD-002",
			Quantity:    987,
			Revenue:     98700.00,
			OrderCount:  456,
			AvgPrice:    100.00,
			Rank:        2,
		},
		{
			ProductID:   "prod-003",
			ProductName: "Economy Widget",
			SKU:         "EWD-003",
			Quantity:    1234,
			Revenue:     61700.00,
			OrderCount:  523,
			AvgPrice:    50.00,
			Rank:        3,
		},
	}

	paymentMethods := []PaymentMethodData{
		{
			Method:        "credit_card",
			OrderCount:    972,
			Revenue:       245252.27,
			Percentage:    66.7,
			AvgOrderValue: 252.37,
		},
		{
			Method:        "paypal",
			OrderCount:    291,
			Revenue:       78523.04,
			Percentage:    21.3,
			AvgOrderValue: 270.01,
		},
		{
			Method:        "bank_transfer",
			OrderCount:    193,
			Revenue:       44114.94,
			Percentage:    12.0,
			AvgOrderValue: 228.58,
		},
	}

	customerSegments := []CustomerSegmentData{
		{
			Segment:       "premium",
			CustomerCount: 198,
			Revenue:       183945.13,
			OrderCount:    287,
			AvgOrderValue: 640.72,
			Percentage:    50.0,
		},
		{
			Segment:       "regular",
			CustomerCount: 567,
			Revenue:       128661.59,
			OrderCount:    734,
			AvgOrderValue: 175.29,
			Percentage:    35.0,
		},
		{
			Segment:       "new",
			CustomerCount: 289,
			Revenue:       55283.53,
			OrderCount:    435,
			AvgOrderValue: 127.01,
			Percentage:    15.0,
		},
	}

	report := &SalesReportData{
		Period:            fmt.Sprintf("Monthly - %s %d", time.Month(month).String(), year),
		StartDate:         startDate,
		EndDate:           endDate,
		TotalRevenue:      totalRevenue,
		TotalOrders:       totalOrders,
		AverageOrderValue: avgOrderValue,
		TopProducts:       topProducts,
		SalesByDay:        salesByDay,
		PaymentMethods:    paymentMethods,
		CustomerSegments:  customerSegments,
		Summary: map[string]interface{}{
			"best_day_of_week":     "Tuesday",
			"conversion_rate":      0.087, // 8.7%
			"repeat_customer_rate": 0.756, // 75.6%
			"growth_rate":          0.154, // 15.4% compared to previous month
		},
	}

	return report, nil
}

// generateTopProductsReport generates a top products report
func (srg *SalesReportGenerator) generateTopProductsReport(ctx context.Context, params map[string]interface{}) (*TopProductsReportData, error) {
	// Parse parameters
	limit := 20
	if limitParam, ok := params["limit"].(float64); ok {
		limit = int(limitParam)
	}

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

	srg.logger.Debug("Generating top products report",
		"limit", limit,
		"period", period,
		"start_date", startDate,
		"end_date", endDate)

	// Generate mock top products data
	topProducts := []ProductSalesData{
		{ProductID: "prod-001", ProductName: "Premium Widget", SKU: "PWD-001", Quantity: 856, Revenue: 188320.00, OrderCount: 428, AvgPrice: 220.00, Rank: 1},
		{ProductID: "prod-002", ProductName: "Standard Widget", SKU: "SWD-002", Quantity: 1243, Revenue: 124300.00, OrderCount: 567, AvgPrice: 100.00, Rank: 2},
		{ProductID: "prod-003", ProductName: "Economy Widget", SKU: "EWD-003", Quantity: 1876, Revenue: 93800.00, OrderCount: 723, AvgPrice: 50.00, Rank: 3},
		{ProductID: "prod-004", ProductName: "Deluxe Gadget", SKU: "DLX-004", Quantity: 234, Revenue: 70200.00, OrderCount: 156, AvgPrice: 300.00, Rank: 4},
		{ProductID: "prod-005", ProductName: "Basic Tool", SKU: "BSC-005", Quantity: 567, Revenue: 56700.00, OrderCount: 234, AvgPrice: 100.00, Rank: 5},
		{ProductID: "prod-006", ProductName: "Professional Kit", SKU: "PRO-006", Quantity: 89, Revenue: 44500.00, OrderCount: 67, AvgPrice: 500.00, Rank: 6},
		{ProductID: "prod-007", ProductName: "Student Package", SKU: "STU-007", Quantity: 345, Revenue: 34500.00, OrderCount: 189, AvgPrice: 100.00, Rank: 7},
		{ProductID: "prod-008", ProductName: "Enterprise Solution", SKU: "ENT-008", Quantity: 23, Revenue: 23000.00, OrderCount: 12, AvgPrice: 1000.00, Rank: 8},
		{ProductID: "prod-009", ProductName: "Starter Set", SKU: "STA-009", Quantity: 456, Revenue: 22800.00, OrderCount: 234, AvgPrice: 50.00, Rank: 9},
		{ProductID: "prod-010", ProductName: "Advanced Module", SKU: "ADV-010", Quantity: 67, Revenue: 20100.00, OrderCount: 45, AvgPrice: 300.00, Rank: 10},
	}

	// Limit results
	if limit < len(topProducts) {
		topProducts = topProducts[:limit]
	}

	categories := []CategoryData{
		{CategoryName: "Widgets", ProductCount: 3, Revenue: 406420.00, OrderCount: 1718, AvgPrice: 147.83, Percentage: 60.5},
		{CategoryName: "Gadgets", ProductCount: 2, Revenue: 114700.00, OrderCount: 223, AvgPrice: 285.71, Percentage: 17.1},
		{CategoryName: "Tools", ProductCount: 2, Revenue: 79200.00, OrderCount: 301, AvgPrice: 166.67, Percentage: 11.8},
		{CategoryName: "Kits", ProductCount: 2, Revenue: 57500.00, OrderCount: 134, AvgPrice: 333.33, Percentage: 8.6},
		{CategoryName: "Solutions", ProductCount: 1, Revenue: 23000.00, OrderCount: 12, AvgPrice: 1000.00, Percentage: 3.4},
	}

	report := &TopProductsReportData{
		Period:      fmt.Sprintf("Top Products - %s", period),
		StartDate:   startDate,
		EndDate:     endDate,
		TopProducts: topProducts,
		Categories:  categories,
		Summary: map[string]interface{}{
			"total_products_analyzed": 247,
			"top_category":            "Widgets",
			"avg_price_range":         "$50 - $1000",
			"best_performer":          topProducts[0].ProductName,
		},
	}

	return report, nil
}

// generateRevenueReport generates a revenue analytics report
func (srg *SalesReportGenerator) generateRevenueReport(ctx context.Context, params map[string]interface{}) (*SalesReportData, error) {
	// This is similar to sales report but focuses more on revenue analytics
	return srg.generateMonthlySalesReport(ctx, params)
}

// getRowCount estimates the number of rows in the report data
func (srg *SalesReportGenerator) getRowCount(data interface{}) int {
	switch reportData := data.(type) {
	case *SalesReportData:
		return len(reportData.SalesByDay) + len(reportData.TopProducts) + len(reportData.PaymentMethods) + len(reportData.CustomerSegments)
	case *TopProductsReportData:
		return len(reportData.TopProducts) + len(reportData.Categories)
	default:
		return 0
	}
}
