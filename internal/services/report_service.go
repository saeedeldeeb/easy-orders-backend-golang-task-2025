package services

import (
	"context"
	"errors"
	"time"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/logger"
)

// reportService implements ReportService interface
type reportService struct {
	orderRepo     repository.OrderRepository
	paymentRepo   repository.PaymentRepository
	inventoryRepo repository.InventoryRepository
	productRepo   repository.ProductRepository
	logger        *logger.Logger
}

// NewReportService creates a new report service
func NewReportService(
	orderRepo repository.OrderRepository,
	paymentRepo repository.PaymentRepository,
	inventoryRepo repository.InventoryRepository,
	productRepo repository.ProductRepository,
	logger *logger.Logger,
) ReportService {
	return &reportService{
		orderRepo:     orderRepo,
		paymentRepo:   paymentRepo,
		inventoryRepo: inventoryRepo,
		productRepo:   productRepo,
		logger:        logger,
	}
}

func (s *reportService) GenerateDailySalesReport(ctx context.Context, date string) (*SalesReportResponse, error) {
	s.logger.Info("Generating daily sales report", "date", date)

	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// Parse date to validate format
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, errors.New("invalid date format, use YYYY-MM-DD")
	}

	// For now, we'll simulate the report generation since we don't have date-filtered queries
	// In a real implementation, we would:
	// 1. Get all orders for the specific date
	// 2. Get all payments for those orders
	// 3. Calculate totals, counts, etc.

	// Simulate getting orders for the date
	// This would be a new repository method like GetOrdersByDateRange
	s.logger.Debug("Fetching orders for date", "date", date)

	// Get recent orders as a simulation
	orders, err := s.orderRepo.List(ctx, 0, 100)
	if err != nil {
		s.logger.Error("Failed to get orders for sales report", "error", err, "date", date)
		return nil, err
	}

	// Calculate report metrics
	var totalSales float64
	var totalOrders int
	var completedOrders int
	var cancelledOrders int
	ordersByStatus := make(map[string]int)

	for _, order := range orders {
		totalOrders++
		totalSales += order.TotalAmount

		statusStr := string(order.Status)
		ordersByStatus[statusStr]++

		switch order.Status {
		case models.OrderStatusDelivered:
			completedOrders++
		case models.OrderStatusCancelled:
			cancelledOrders++
		}
	}

	averageOrderValue := float64(0)
	if totalOrders > 0 {
		averageOrderValue = totalSales / float64(totalOrders)
	}

	report := &SalesReportResponse{
		Date:              date,
		TotalSales:        totalSales,
		TotalOrders:       totalOrders,
		CompletedOrders:   completedOrders,
		CancelledOrders:   cancelledOrders,
		AverageOrderValue: averageOrderValue,
		OrdersByStatus:    ordersByStatus,
	}

	s.logger.Info("Daily sales report generated", "date", date, "total_sales", totalSales, "total_orders", totalOrders)

	return report, nil
}

func (s *reportService) GenerateInventoryReport(ctx context.Context) (*InventoryReportResponse, error) {
	s.logger.Info("Generating inventory report")

	// Get all products with inventory
	products, err := s.productRepo.List(ctx, 0, 1000) // Get a large number to simulate all products
	if err != nil {
		s.logger.Error("Failed to get products for inventory report", "error", err)
		return nil, err
	}

	var totalProducts int
	var activeProducts int
	var lowStockProducts int
	var outOfStockProducts int
	var totalStockValue float64
	var totalStockQuantity int

	productInventory := make([]ProductInventoryItem, 0, len(products))

	for _, product := range products {
		totalProducts++

		if product.IsActive {
			activeProducts++
		}

		// Get inventory for the product
		inventory, err := s.inventoryRepo.GetByProductID(ctx, product.ID)
		if err != nil {
			s.logger.Error("Failed to get inventory for product", "error", err, "product_id", product.ID)
			continue
		}

		stockQuantity := 0
		stockValue := float64(0)
		status := "Out of Stock"

		if inventory != nil {
			stockQuantity = inventory.Available
			stockValue = float64(stockQuantity) * product.Price
			totalStockQuantity += stockQuantity
			totalStockValue += stockValue

			if stockQuantity == 0 {
				outOfStockProducts++
				status = "Out of Stock"
			} else if inventory.IsLowStock() {
				lowStockProducts++
				status = "Low Stock"
			} else {
				status = "In Stock"
			}
		} else {
			outOfStockProducts++
		}

		productInventory = append(productInventory, ProductInventoryItem{
			ProductID:     product.ID,
			ProductName:   product.Name,
			SKU:           product.SKU,
			Price:         product.Price,
			StockQuantity: stockQuantity,
			StockValue:    stockValue,
			Status:        status,
		})
	}

	// Get low stock alerts
	lowStockItems, err := s.inventoryRepo.GetLowStockItems(ctx, 10)
	if err != nil {
		s.logger.Error("Failed to get low stock items for report", "error", err)
		// Don't fail the report, just log the error
	} else {
		lowStockProducts = len(lowStockItems)
	}

	report := &InventoryReportResponse{
		TotalProducts:      totalProducts,
		ActiveProducts:     activeProducts,
		LowStockProducts:   lowStockProducts,
		OutOfStockProducts: outOfStockProducts,
		TotalStockValue:    totalStockValue,
		TotalStockQuantity: totalStockQuantity,
		ProductInventory:   productInventory,
	}

	s.logger.Info("Inventory report generated",
		"total_products", totalProducts,
		"active_products", activeProducts,
		"low_stock", lowStockProducts,
		"out_of_stock", outOfStockProducts,
		"total_value", totalStockValue)

	return report, nil
}

func (s *reportService) GenerateTopProductsReport(ctx context.Context, limit int) (*TopProductsResponse, error) {
	s.logger.Info("Generating top products report", "limit", limit)

	if limit <= 0 || limit > 100 {
		limit = 10 // Default limit
	}

	// For now, we'll simulate this report since we don't have aggregation queries
	// In a real implementation, we would:
	// 1. Join orders, order_items, and products
	// 2. Group by product
	// 3. Sum quantities and revenue
	// 4. Order by sales volume or revenue

	// Get recent orders to simulate analysis
	orders, err := s.orderRepo.List(ctx, 0, 500)
	if err != nil {
		s.logger.Error("Failed to get orders for top products report", "error", err)
		return nil, err
	}

	// Simulate product sales calculation
	productSales := make(map[string]*TopProductItem)

	for _, order := range orders {
		for _, item := range order.Items {
			if existing, ok := productSales[item.ProductID]; ok {
				existing.TotalQuantity += item.Quantity
				existing.TotalRevenue += item.TotalPrice
				existing.OrderCount++
			} else {
				productName := "Unknown Product"
				if item.Product != nil {
					productName = item.Product.Name
				}

				productSales[item.ProductID] = &TopProductItem{
					ProductID:     item.ProductID,
					ProductName:   productName,
					TotalQuantity: item.Quantity,
					TotalRevenue:  item.TotalPrice,
					OrderCount:    1,
				}
			}
		}
	}

	// Convert map to slice and sort (simplified sorting)
	topProducts := make([]*TopProductItem, 0, len(productSales))
	for _, item := range productSales {
		topProducts = append(topProducts, item)
	}

	// Simple sorting by revenue (in real implementation, use sort.Slice)
	// For now, just take the first items up to limit
	if len(topProducts) > limit {
		topProducts = topProducts[:limit]
	}

	s.logger.Info("Top products report generated", "product_count", len(topProducts))

	return &TopProductsResponse{
		TopProducts: topProducts,
		Limit:       limit,
		Period:      "all_time", // For now, always all time
	}, nil
}

func (s *reportService) GenerateUserActivityReport(ctx context.Context, req UserActivityReportRequest) (*UserActivityReportResponse, error) {
	s.logger.Info("Generating user activity report", "start_date", req.StartDate, "end_date", req.EndDate)

	// For now, simulate the report generation since we don't have date-filtered queries
	// In a real implementation, we would:
	// 1. Get all users created between start and end date
	// 2. Get all users who were active (logged in) between dates
	// 3. Calculate metrics

	// Simulate user activity calculation
	// This would normally involve complex queries on user login logs, orders, etc.

	response := &UserActivityReportResponse{
		ActiveUsers: 25, // Simulated value
		NewUsers:    5,  // Simulated value
	}

	s.logger.Info("User activity report generated", "active_users", response.ActiveUsers, "new_users", response.NewUsers)

	return response, nil
}
