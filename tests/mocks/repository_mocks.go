package mocks

import (
	"context"
	"time"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/repository"

	"github.com/stretchr/testify/mock"
)

// MockProductRepository is a mock implementation of repository.ProductRepository
type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) Create(ctx context.Context, product *models.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockProductRepository) CreateWithInventory(ctx context.Context, product *models.Product, inventory *models.Inventory) error {
	args := m.Called(ctx, product, inventory)
	return args.Error(0)
}

func (m *MockProductRepository) GetByID(ctx context.Context, id string) (*models.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductRepository) GetBySKU(ctx context.Context, sku string) (*models.Product, error) {
	args := m.Called(ctx, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductRepository) Update(ctx context.Context, product *models.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockProductRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductRepository) List(ctx context.Context, offset, limit int) ([]*models.Product, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Product), args.Error(1)
}

func (m *MockProductRepository) Search(ctx context.Context, query string, offset, limit int) ([]*models.Product, error) {
	args := m.Called(ctx, query, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Product), args.Error(1)
}

func (m *MockProductRepository) GetActive(ctx context.Context, offset, limit int) ([]*models.Product, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Product), args.Error(1)
}

func (m *MockProductRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductRepository) CountActive(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProductRepository) CountSearch(ctx context.Context, query string) (int64, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(int64), args.Error(1)
}

// MockInventoryRepository is a mock implementation of repository.InventoryRepository
type MockInventoryRepository struct {
	mock.Mock
}

func (m *MockInventoryRepository) Create(ctx context.Context, inventory *models.Inventory) error {
	args := m.Called(ctx, inventory)
	return args.Error(0)
}

func (m *MockInventoryRepository) GetByProductID(ctx context.Context, productID string) (*models.Inventory, error) {
	args := m.Called(ctx, productID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) UpdateStock(ctx context.Context, productID string, quantity int) error {
	args := m.Called(ctx, productID, quantity)
	return args.Error(0)
}

func (m *MockInventoryRepository) ReserveStock(ctx context.Context, productID string, quantity int) error {
	args := m.Called(ctx, productID, quantity)
	return args.Error(0)
}

func (m *MockInventoryRepository) ReleaseStock(ctx context.Context, productID string, quantity int) error {
	args := m.Called(ctx, productID, quantity)
	return args.Error(0)
}

func (m *MockInventoryRepository) FulfillStock(ctx context.Context, productID string, quantity int) error {
	args := m.Called(ctx, productID, quantity)
	return args.Error(0)
}

func (m *MockInventoryRepository) GetLowStockItems(ctx context.Context, threshold int) ([]*models.Inventory, error) {
	args := m.Called(ctx, threshold)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Inventory), args.Error(1)
}

func (m *MockInventoryRepository) BulkReserve(ctx context.Context, items []repository.InventoryReservation) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

func (m *MockInventoryRepository) BulkRelease(ctx context.Context, items []repository.InventoryReservation) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

// MockOrderRepository is a mock implementation of repository.OrderRepository
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) Create(ctx context.Context, order *models.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) GetByID(ctx context.Context, id string) (*models.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderRepository) GetByIDWithItems(ctx context.Context, id string) (*models.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockOrderRepository) GetByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.Order, error) {
	args := m.Called(ctx, userID, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Order), args.Error(1)
}

func (m *MockOrderRepository) Update(ctx context.Context, order *models.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) UpdateStatus(ctx context.Context, id string, status models.OrderStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockOrderRepository) List(ctx context.Context, offset, limit int) ([]*models.Order, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Order), args.Error(1)
}

func (m *MockOrderRepository) ListByStatus(ctx context.Context, status models.OrderStatus, offset, limit int) ([]*models.Order, error) {
	args := m.Called(ctx, status, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Order), args.Error(1)
}

func (m *MockOrderRepository) GetByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*models.Order, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Order), args.Error(1)
}

func (m *MockOrderRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockOrderRepository) CountByStatus(ctx context.Context, status models.OrderStatus) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockOrderRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

// MockOrderItemRepository is a mock implementation of repository.OrderItemRepository
type MockOrderItemRepository struct {
	mock.Mock
}

func (m *MockOrderItemRepository) Create(ctx context.Context, item *models.OrderItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockOrderItemRepository) CreateBatch(ctx context.Context, items []*models.OrderItem) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

func (m *MockOrderItemRepository) GetByOrderID(ctx context.Context, orderID string) ([]*models.OrderItem, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.OrderItem), args.Error(1)
}

func (m *MockOrderItemRepository) Update(ctx context.Context, item *models.OrderItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockOrderItemRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockUserRepository is a mock implementation of repository.UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int) ([]*models.User, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// MockPaymentRepository is a mock implementation of repository.PaymentRepository
type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetByID(ctx context.Context, id string) (*models.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}

func (m *MockPaymentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}

func (m *MockPaymentRepository) GetByOrderID(ctx context.Context, orderID string) ([]*models.Payment, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Payment), args.Error(1)
}

func (m *MockPaymentRepository) Update(ctx context.Context, payment *models.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) UpdateStatus(ctx context.Context, id string, status models.PaymentStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockPaymentRepository) List(ctx context.Context, offset, limit int) ([]*models.Payment, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Payment), args.Error(1)
}
