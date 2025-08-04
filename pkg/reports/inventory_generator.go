package reports

import (
	"context"
	"fmt"
	"time"

	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/logger"
)

// InventoryReportGenerator generates inventory-related reports
type InventoryReportGenerator struct {
	inventoryRepo repository.InventoryRepository
	productRepo   repository.ProductRepository
	orderRepo     repository.OrderRepository
	logger        *logger.Logger
}

// NewInventoryReportGenerator creates a new inventory report generator
func NewInventoryReportGenerator(
	inventoryRepo repository.InventoryRepository,
	productRepo repository.ProductRepository,
	orderRepo repository.OrderRepository,
	logger *logger.Logger,
) *InventoryReportGenerator {
	return &InventoryReportGenerator{
		inventoryRepo: inventoryRepo,
		productRepo:   productRepo,
		orderRepo:     orderRepo,
		logger:        logger,
	}
}

// GenerateReport generates an inventory report based on the request
func (irg *InventoryReportGenerator) GenerateReport(ctx context.Context, req *ReportRequest) (*ReportResult, error) {
	irg.logger.Info("Generating inventory report",
		"type", string(req.Type),
		"id", req.ID)

	startTime := time.Now()

	var reportData interface{}
	var err error

	switch req.Type {
	case ReportTypeLowStock:
		reportData, err = irg.generateLowStockReport(ctx, req.Parameters)
	case ReportTypeInventoryValue:
		reportData, err = irg.generateInventoryValueReport(ctx, req.Parameters)
	default:
		return nil, fmt.Errorf("unsupported inventory report type: %s", req.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate inventory report: %w", err)
	}

	processingTime := time.Since(startTime)

	result := &ReportResult{
		ID:             fmt.Sprintf("inventory_%s_%d", req.ID, time.Now().UnixNano()),
		RequestID:      req.ID,
		Type:           req.Type,
		Format:         req.Format,
		Status:         ReportStatusCompleted,
		Data:           reportData,
		ProcessingTime: processingTime,
		RowCount:       irg.getRowCount(reportData),
		Metadata: map[string]interface{}{
			"generator":     "InventoryReportGenerator",
			"processing_ms": processingTime.Milliseconds(),
			"generated_at":  time.Now(),
		},
	}

	irg.logger.Info("Inventory report generated successfully",
		"type", string(req.Type),
		"id", req.ID,
		"processing_time_ms", processingTime.Milliseconds(),
		"row_count", result.RowCount)

	return result, nil
}

// GetSupportedTypes returns the report types this generator supports
func (irg *InventoryReportGenerator) GetSupportedTypes() []ReportType {
	return []ReportType{
		ReportTypeLowStock,
		ReportTypeInventoryValue,
	}
}

// GetName returns the generator name
func (irg *InventoryReportGenerator) GetName() string {
	return "InventoryReportGenerator"
}

// EstimateGenerationTime estimates how long the report will take to generate
func (irg *InventoryReportGenerator) EstimateGenerationTime(req *ReportRequest) time.Duration {
	switch req.Type {
	case ReportTypeLowStock:
		return 1 * time.Second
	case ReportTypeInventoryValue:
		return 3 * time.Second
	default:
		return 2 * time.Second
	}
}

// generateLowStockReport generates a low stock alert report
func (irg *InventoryReportGenerator) generateLowStockReport(ctx context.Context, params map[string]interface{}) (*LowStockReportData, error) {
	// Parse parameters
	threshold := 10
	if thresholdParam, ok := params["threshold"].(float64); ok {
		threshold = int(thresholdParam)
	}

	irg.logger.Debug("Generating low stock report", "threshold", threshold)

	// In a real implementation, this would query the database
	// For now, we'll simulate low stock items

	now := time.Now()
	lastWeek := now.AddDate(0, 0, -7)
	lastMonth := now.AddDate(0, -1, 0)

	lowStockItems := []LowStockItemData{
		{
			ProductID:        "prod-001",
			ProductName:      "Premium Widget",
			SKU:              "PWD-001",
			CurrentStock:     3,
			ReservedStock:    2,
			AvailableStock:   1,
			MinThreshold:     10,
			StockLevel:       "critical",
			LastRestocked:    &lastMonth,
			DaysOutOfStock:   0,
			EstimatedDemand:  15,
			RecommendedOrder: 50,
		},
		{
			ProductID:        "prod-005",
			ProductName:      "Basic Tool",
			SKU:              "BSC-005",
			CurrentStock:     5,
			ReservedStock:    1,
			AvailableStock:   4,
			MinThreshold:     15,
			StockLevel:       "critical",
			LastRestocked:    &lastWeek,
			DaysOutOfStock:   0,
			EstimatedDemand:  20,
			RecommendedOrder: 75,
		},
		{
			ProductID:        "prod-007",
			ProductName:      "Student Package",
			SKU:              "STU-007",
			CurrentStock:     8,
			ReservedStock:    2,
			AvailableStock:   6,
			MinThreshold:     12,
			StockLevel:       "warning",
			LastRestocked:    &lastWeek,
			DaysOutOfStock:   0,
			EstimatedDemand:  18,
			RecommendedOrder: 60,
		},
		{
			ProductID:        "prod-009",
			ProductName:      "Starter Set",
			SKU:              "STA-009",
			CurrentStock:     0,
			ReservedStock:    0,
			AvailableStock:   0,
			MinThreshold:     20,
			StockLevel:       "critical",
			LastRestocked:    &lastMonth,
			DaysOutOfStock:   3,
			EstimatedDemand:  25,
			RecommendedOrder: 100,
		},
		{
			ProductID:        "prod-012",
			ProductName:      "Replacement Parts",
			SKU:              "RPL-012",
			CurrentStock:     7,
			ReservedStock:    3,
			AvailableStock:   4,
			MinThreshold:     15,
			StockLevel:       "warning",
			LastRestocked:    &lastWeek,
			DaysOutOfStock:   0,
			EstimatedDemand:  12,
			RecommendedOrder: 40,
		},
		{
			ProductID:        "prod-015",
			ProductName:      "Maintenance Kit",
			SKU:              "MNT-015",
			CurrentStock:     2,
			ReservedStock:    1,
			AvailableStock:   1,
			MinThreshold:     8,
			StockLevel:       "critical",
			LastRestocked:    &lastMonth,
			DaysOutOfStock:   0,
			EstimatedDemand:  10,
			RecommendedOrder: 30,
		},
	}

	// Filter by threshold
	var filteredItems []LowStockItemData
	criticalCount := 0
	warningCount := 0

	for _, item := range lowStockItems {
		if item.AvailableStock <= threshold {
			filteredItems = append(filteredItems, item)
			if item.StockLevel == "critical" {
				criticalCount++
			} else {
				warningCount++
			}
		}
	}

	report := &LowStockReportData{
		GeneratedAt:   now,
		Threshold:     threshold,
		TotalItems:    len(filteredItems),
		CriticalItems: criticalCount,
		WarningItems:  warningCount,
		Items:         filteredItems,
		Summary: map[string]interface{}{
			"out_of_stock_count":        1,
			"total_recommended_orders":  355,
			"estimated_total_cost":      "$45,230",
			"most_critical_product":     "Starter Set",
			"longest_out_of_stock_days": 3,
			"categories_affected": []string{
				"Widgets", "Tools", "Packages", "Kits", "Parts",
			},
		},
	}

	return report, nil
}

// generateInventoryValueReport generates an inventory valuation report
func (irg *InventoryReportGenerator) generateInventoryValueReport(ctx context.Context, params map[string]interface{}) (*InventoryValueReportData, error) {
	irg.logger.Debug("Generating inventory value report")

	// In a real implementation, this would calculate actual inventory values
	// For now, we'll simulate inventory value data

	now := time.Now()

	// Simulate inventory items with values
	items := []InventoryValueItem{
		{
			ProductID:     "prod-001",
			ProductName:   "Premium Widget",
			SKU:           "PWD-001",
			Quantity:      45,
			UnitCost:      180.00,
			UnitPrice:     220.00,
			TotalValue:    8100.00,
			TotalRetail:   9900.00,
			Margin:        1800.00,
			MarginPercent: 18.18,
			TurnoverRate:  0.75,
			Category:      "Widgets",
		},
		{
			ProductID:     "prod-002",
			ProductName:   "Standard Widget",
			SKU:           "SWD-002",
			Quantity:      123,
			UnitCost:      75.00,
			UnitPrice:     100.00,
			TotalValue:    9225.00,
			TotalRetail:   12300.00,
			Margin:        3075.00,
			MarginPercent: 25.00,
			TurnoverRate:  1.20,
			Category:      "Widgets",
		},
		{
			ProductID:     "prod-003",
			ProductName:   "Economy Widget",
			SKU:           "EWD-003",
			Quantity:      234,
			UnitCost:      35.00,
			UnitPrice:     50.00,
			TotalValue:    8190.00,
			TotalRetail:   11700.00,
			Margin:        3510.00,
			MarginPercent: 30.00,
			TurnoverRate:  2.10,
			Category:      "Widgets",
		},
		{
			ProductID:     "prod-004",
			ProductName:   "Deluxe Gadget",
			SKU:           "DLX-004",
			Quantity:      67,
			UnitCost:      240.00,
			UnitPrice:     300.00,
			TotalValue:    16080.00,
			TotalRetail:   20100.00,
			Margin:        4020.00,
			MarginPercent: 20.00,
			TurnoverRate:  0.45,
			Category:      "Gadgets",
		},
		{
			ProductID:     "prod-005",
			ProductName:   "Basic Tool",
			SKU:           "BSC-005",
			Quantity:      5,
			UnitCost:      80.00,
			UnitPrice:     100.00,
			TotalValue:    400.00,
			TotalRetail:   500.00,
			Margin:        100.00,
			MarginPercent: 20.00,
			TurnoverRate:  3.50,
			Category:      "Tools",
		},
	}

	// Calculate totals
	var totalValue, totalRetail, totalMargin float64
	var totalQuantity int
	categoryTotals := make(map[string]*CategoryValue)

	for _, item := range items {
		totalValue += item.TotalValue
		totalRetail += item.TotalRetail
		totalMargin += item.Margin
		totalQuantity += item.Quantity

		if categoryTotals[item.Category] == nil {
			categoryTotals[item.Category] = &CategoryValue{
				Category: item.Category,
			}
		}
		cat := categoryTotals[item.Category]
		cat.ProductCount++
		cat.TotalQuantity += item.Quantity
		cat.TotalValue += item.TotalValue
		cat.TotalRetail += item.TotalRetail
		cat.TotalMargin += item.Margin
	}

	// Convert category map to slice
	var categories []CategoryValue
	for _, cat := range categoryTotals {
		cat.MarginPercent = (cat.TotalMargin / cat.TotalValue) * 100
		cat.Percentage = (cat.TotalValue / totalValue) * 100
		categories = append(categories, *cat)
	}

	report := &InventoryValueReportData{
		GeneratedAt:      now,
		TotalItems:       len(items),
		TotalQuantity:    totalQuantity,
		TotalValue:       totalValue,
		TotalRetailValue: totalRetail,
		TotalMargin:      totalMargin,
		MarginPercent:    (totalMargin / totalValue) * 100,
		Items:            items,
		Categories:       categories,
		Summary: map[string]interface{}{
			"highest_value_item":     "Deluxe Gadget",
			"highest_margin_item":    "Economy Widget",
			"fastest_turnover":       "Basic Tool",
			"slowest_turnover":       "Deluxe Gadget",
			"avg_turnover_rate":      1.4,
			"total_dead_stock_value": "$2,450",
		},
	}

	return report, nil
}

// getRowCount estimates the number of rows in the report data
func (irg *InventoryReportGenerator) getRowCount(data interface{}) int {
	switch reportData := data.(type) {
	case *LowStockReportData:
		return len(reportData.Items)
	case *InventoryValueReportData:
		return len(reportData.Items) + len(reportData.Categories)
	default:
		return 0
	}
}

// InventoryValueReportData represents inventory valuation report data
type InventoryValueReportData struct {
	GeneratedAt      time.Time              `json:"generated_at"`
	TotalItems       int                    `json:"total_items"`
	TotalQuantity    int                    `json:"total_quantity"`
	TotalValue       float64                `json:"total_value"`
	TotalRetailValue float64                `json:"total_retail_value"`
	TotalMargin      float64                `json:"total_margin"`
	MarginPercent    float64                `json:"margin_percent"`
	Items            []InventoryValueItem   `json:"items"`
	Categories       []CategoryValue        `json:"categories"`
	Summary          map[string]interface{} `json:"summary"`
}

// InventoryValueItem represents individual inventory item valuation
type InventoryValueItem struct {
	ProductID     string  `json:"product_id"`
	ProductName   string  `json:"product_name"`
	SKU           string  `json:"sku"`
	Quantity      int     `json:"quantity"`
	UnitCost      float64 `json:"unit_cost"`
	UnitPrice     float64 `json:"unit_price"`
	TotalValue    float64 `json:"total_value"`
	TotalRetail   float64 `json:"total_retail"`
	Margin        float64 `json:"margin"`
	MarginPercent float64 `json:"margin_percent"`
	TurnoverRate  float64 `json:"turnover_rate"`
	Category      string  `json:"category"`
}

// CategoryValue represents category-level inventory valuation
type CategoryValue struct {
	Category      string  `json:"category"`
	ProductCount  int     `json:"product_count"`
	TotalQuantity int     `json:"total_quantity"`
	TotalValue    float64 `json:"total_value"`
	TotalRetail   float64 `json:"total_retail"`
	TotalMargin   float64 `json:"total_margin"`
	MarginPercent float64 `json:"margin_percent"`
	Percentage    float64 `json:"percentage"`
}
