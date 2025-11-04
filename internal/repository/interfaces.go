package repository

import (
	"context"

	"easy-orders-backend/internal/models"
)

// Repository interfaces define contracts for data access layer
// These will be implemented by concrete repository structs

// UserRepository defines user data access methods
type UserRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*models.User, error)
	Count(ctx context.Context) (int64, error)
}

// ProductRepository defines product data access methods
type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, id string) (*models.Product, error)
	GetBySKU(ctx context.Context, sku string) (*models.Product, error)
	Update(ctx context.Context, product *models.Product) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*models.Product, error)
	Search(ctx context.Context, query string, offset, limit int) ([]*models.Product, error)
	GetActive(ctx context.Context, offset, limit int) ([]*models.Product, error)
	Count(ctx context.Context) (int64, error)
	CountActive(ctx context.Context) (int64, error)
	CountSearch(ctx context.Context, query string) (int64, error)
}

// OrderRepository defines order data access methods
type OrderRepository interface {
	Create(ctx context.Context, order *models.Order) error
	GetByID(ctx context.Context, id string) (*models.Order, error)
	GetByIDWithItems(ctx context.Context, id string) (*models.Order, error)
	GetByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.Order, error)
	Update(ctx context.Context, order *models.Order) error
	UpdateStatus(ctx context.Context, id string, status models.OrderStatus) error
	List(ctx context.Context, offset, limit int) ([]*models.Order, error)
	ListByStatus(ctx context.Context, status models.OrderStatus, offset, limit int) ([]*models.Order, error)
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status models.OrderStatus) (int64, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
}

// OrderItemRepository defines order item data access methods
type OrderItemRepository interface {
	Create(ctx context.Context, item *models.OrderItem) error
	CreateBatch(ctx context.Context, items []*models.OrderItem) error
	GetByOrderID(ctx context.Context, orderID string) ([]*models.OrderItem, error)
	Update(ctx context.Context, item *models.OrderItem) error
	Delete(ctx context.Context, id string) error
}

// InventoryRepository defines inventory data access methods
type InventoryRepository interface {
	GetByProductID(ctx context.Context, productID string) (*models.Inventory, error)
	UpdateStock(ctx context.Context, productID string, quantity int) error
	ReserveStock(ctx context.Context, productID string, quantity int) error
	ReleaseStock(ctx context.Context, productID string, quantity int) error
	FulfillStock(ctx context.Context, productID string, quantity int) error
	GetLowStockItems(ctx context.Context, threshold int) ([]*models.Inventory, error)
	BulkReserve(ctx context.Context, items []InventoryReservation) error
	BulkRelease(ctx context.Context, items []InventoryReservation) error
}

// InventoryReservation represents a stock reservation request
type InventoryReservation struct {
	ProductID string
	Quantity  int
}

// PaymentRepository defines payment data access methods
type PaymentRepository interface {
	Create(ctx context.Context, payment *models.Payment) error
	GetByID(ctx context.Context, id string) (*models.Payment, error)
	GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error)
	GetByOrderID(ctx context.Context, orderID string) ([]*models.Payment, error)
	Update(ctx context.Context, payment *models.Payment) error
	UpdateStatus(ctx context.Context, id string, status models.PaymentStatus) error
	List(ctx context.Context, offset, limit int) ([]*models.Payment, error)
}

// NotificationRepository defines notification data access methods
type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	GetByID(ctx context.Context, id string) (*models.Notification, error)
	GetByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.Notification, error)
	GetUnreadByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.Notification, error)
	MarkAsRead(ctx context.Context, id string) error
	MarkAllAsRead(ctx context.Context, userID string) error
	GetUnreadCount(ctx context.Context, userID string) (int, error)
	Delete(ctx context.Context, id string) error
}

// AuditLogRepository defines audit log data access methods
type AuditLogRepository interface {
	Create(ctx context.Context, log *models.AuditLog) error
	GetByID(ctx context.Context, id string) (*models.AuditLog, error)
	GetByEntityID(ctx context.Context, entityType, entityID string, offset, limit int) ([]*models.AuditLog, error)
	GetByUserID(ctx context.Context, userID string, offset, limit int) ([]*models.AuditLog, error)
	List(ctx context.Context, offset, limit int) ([]*models.AuditLog, error)
}
