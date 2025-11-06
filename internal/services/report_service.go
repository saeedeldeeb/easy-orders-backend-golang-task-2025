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

	// Parse date to validate format and create date range
	startDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, errors.New("invalid date format, use YYYY-MM-DD")
	}

	// Create an end date (next day at 00:00:00) for exclusive range
	endDate := startDate.AddDate(0, 0, 1)

	s.logger.Debug("Fetching orders for date range", "date", date, "start_date", startDate, "end_date", endDate)

	// Get all orders for the specific date using a date range query
	orders, err := s.orderRepo.GetByDateRange(ctx, startDate, endDate)
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

		statusStr := string(order.Status)
		ordersByStatus[statusStr]++

		switch order.Status {
		case models.OrderStatusDelivered:
			completedOrders++
			// Only count completed/delivered orders in total sales
			totalSales += order.TotalAmount
		case models.OrderStatusCancelled:
			cancelledOrders++
		}
	}

	// Calculate the average order value based on completed orders only
	averageOrderValue := float64(0)
	if completedOrders > 0 {
		averageOrderValue = totalSales / float64(completedOrders)
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
