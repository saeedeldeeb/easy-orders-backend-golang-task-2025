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
