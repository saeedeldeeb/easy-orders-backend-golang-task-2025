package services

import (
	"context"
	"time"

	"easy-orders-backend/internal/models"
)

// Service interfaces define business logic contracts
// These will be implemented by concrete service structs

// UserService defines user business logic
type UserService interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error)
	GetUser(ctx context.Context, id string) (*UserResponse, error)
	UpdateUser(ctx context.Context, id string, req UpdateUserRequest) (*UserResponse, error)
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, req ListUsersRequest) (*ListUsersResponse, error)
	AuthenticateUser(ctx context.Context, email, password string) (*AuthResponse, error)
}

// ProductService defines product business logic
type ProductService interface {
	CreateProduct(ctx context.Context, req CreateProductRequest) (*ProductResponse, error)
	GetProduct(ctx context.Context, id string) (*ProductResponse, error)
	UpdateProduct(ctx context.Context, id string, req UpdateProductRequest) (*ProductResponse, error)
	DeleteProduct(ctx context.Context, id string) error
	ListProducts(ctx context.Context, req ListProductsRequest) (*ListProductsResponse, error)
	SearchProducts(ctx context.Context, req SearchProductsRequest) (*ListProductsResponse, error)
}

// OrderService defines order business logic
type OrderService interface {
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error)
	GetOrder(ctx context.Context, id string) (*OrderResponse, error)
	UpdateOrderStatus(ctx context.Context, id string, status models.OrderStatus) (*OrderResponse, error)
	CancelOrder(ctx context.Context, id string) error
	ListOrders(ctx context.Context, req ListOrdersRequest) (*ListOrdersResponse, error)
	GetUserOrders(ctx context.Context, userID string, req ListOrdersRequest) (*ListOrdersResponse, error)
}

// InventoryService defines inventory business logic
type InventoryService interface {
	CheckAvailability(ctx context.Context, productID string, quantity int) (bool, error)
	ReserveInventory(ctx context.Context, items []InventoryItem) error
	ReleaseInventory(ctx context.Context, items []InventoryItem) error
	UpdateStock(ctx context.Context, productID string, quantity int) error
	GetLowStockAlert(ctx context.Context, threshold int) (*LowStockResponse, error)
}

// EnhancedInventoryService extends InventoryService with advanced concurrency features
type EnhancedInventoryService interface {
	InventoryService
	ReserveInventoryConcurrent(ctx context.Context, items []InventoryItem) error
	CheckAvailabilityWithCache(ctx context.Context, productID string, quantity int) (bool, error)
	ProcessHighVolumeOrders(ctx context.Context, orders []HighVolumeOrder, workerCount int) (*HighVolumeProcessingResult, error)
}

// HighVolumeOrder High-volume processing types
type HighVolumeOrder struct {
	OrderID   string `json:"order_id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type HighVolumeOrderResult struct {
	OrderID        string        `json:"order_id"`
	ProductID      string        `json:"product_id"`
	Quantity       int           `json:"quantity"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`
	ProcessingTime time.Duration `json:"processing_time"`
	Error          error         `json:"error,omitempty"`
}

type HighVolumeProcessingResult struct {
	TotalOrders       int                     `json:"total_orders"`
	SuccessfulOrders  int                     `json:"successful_orders"`
	FailedOrders      int                     `json:"failed_orders"`
	SuccessfulResults []HighVolumeOrderResult `json:"successful_results"`
	FailedResults     []HighVolumeOrderResult `json:"failed_results"`
	ProcessingTime    time.Duration           `json:"processing_time"`
}

// PaymentService defines payment business logic
type PaymentService interface {
	ProcessPayment(ctx context.Context, req ProcessPaymentRequest) (*PaymentResponse, error)
	GetPayment(ctx context.Context, id string) (*PaymentResponse, error)
	RefundPayment(ctx context.Context, id string, amount float64) (*PaymentResponse, error)
	GetOrderPayments(ctx context.Context, orderID string) ([]*PaymentResponse, error)
}

// NotificationService defines notification business logic
type NotificationService interface {
	SendNotification(ctx context.Context, req SendNotificationRequest) error
	GetUserNotifications(ctx context.Context, userID string, req ListNotificationsRequest) (*ListNotificationsResponse, error)
	MarkAsRead(ctx context.Context, notificationID string) error
	GetUnreadCount(ctx context.Context, userID string) (int, error)
}

// ReportService defines reporting business logic
type ReportService interface {
	GenerateDailySalesReport(ctx context.Context, date string) (*SalesReportResponse, error)
	GenerateInventoryReport(ctx context.Context) (*InventoryReportResponse, error)
	GenerateUserActivityReport(ctx context.Context, req UserActivityReportRequest) (*UserActivityReportResponse, error)
}

// OrderPipelineService defines concurrent order processing pipeline logic
type OrderPipelineService interface {
	ProcessOrder(ctx context.Context, req CreateOrderRequest) (*OrderPipelineResult, error)
	ProcessOrderAsync(ctx context.Context, req CreateOrderRequest) (<-chan *OrderPipelineResult, error)
}

// CreateUserRequest Request/Response structs
type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
}

type UpdateUserRequest struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=1"`
	Email *string `json:"email,omitempty" validate:"omitempty,email"`
}

type ListUsersRequest struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type UserResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

type ListUsersResponse struct {
	Users  []*UserResponse `json:"users"`
	Offset int             `json:"offset"`
	Limit  int             `json:"limit"`
	Total  int             `json:"total"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type CreateProductRequest struct {
	Name         string  `json:"name" validate:"required"`
	Description  string  `json:"description"`
	Price        float64 `json:"price" validate:"required,gt=0"`
	SKU          string  `json:"sku" validate:"required"`
	CategoryID   string  `json:"category_id"`
	InitialStock int     `json:"initial_stock,omitempty"`
	MinStock     int     `json:"min_stock,omitempty"`
	MaxStock     int     `json:"max_stock,omitempty"`
}

type UpdateProductRequest struct {
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	Price       float64 `json:"price,omitempty" validate:"omitempty,gt=0"`
	CategoryID  string  `json:"category_id,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type ListProductsRequest struct {
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
	CategoryID string `json:"category_id,omitempty"`
	ActiveOnly bool   `json:"active_only,omitempty"`
}

type SearchProductsRequest struct {
	Query  string `json:"query" validate:"required"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

type ProductResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	SKU         string  `json:"sku"`
	CategoryID  string  `json:"category_id"`
	IsActive    bool    `json:"is_active"`
	Stock       int     `json:"stock"`
}

type ListProductsResponse struct {
	Products []*ProductResponse `json:"products"`
	Offset   int                `json:"offset"`
	Limit    int                `json:"limit"`
	Total    int                `json:"total"`
}

type CreateOrderRequest struct {
	UserID string      `json:"user_id" validate:"required"`
	Items  []OrderItem `json:"items" validate:"required,dive"`
	Notes  string      `json:"notes,omitempty"`
}

type OrderItem struct {
	ProductID string  `json:"product_id" validate:"required"`
	Quantity  int     `json:"quantity" validate:"required,gt=0"`
	UnitPrice float64 `json:"unit_price"`
}

type ListOrdersRequest struct {
	Offset int                `json:"offset"`
	Limit  int                `json:"limit"`
	Status models.OrderStatus `json:"status,omitempty"`
}

type OrderResponse struct {
	ID     string             `json:"id"`
	UserID string             `json:"user_id"`
	Status models.OrderStatus `json:"status"`
	Items  []OrderItem        `json:"items"`
	Total  float64            `json:"total"`
}

type ListOrdersResponse struct {
	Orders []*OrderResponse `json:"orders"`
	Offset int              `json:"offset"`
	Limit  int              `json:"limit"`
	Total  int              `json:"total"`
}

type InventoryItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type ProcessPaymentRequest struct {
	OrderID           string  `json:"order_id" validate:"required"`
	Amount            float64 `json:"amount" validate:"required,gt=0"`
	PaymentType       string  `json:"payment_type" validate:"required"`
	ExternalReference string  `json:"external_reference,omitempty"`
}

type PaymentResponse struct {
	ID      string               `json:"id"`
	OrderID string               `json:"order_id"`
	Amount  float64              `json:"amount"`
	Status  models.PaymentStatus `json:"status"`
}

type SendNotificationRequest struct {
	UserID  string `json:"user_id" validate:"required"`
	Type    string `json:"type,omitempty"`
	Channel string `json:"channel,omitempty"`
	Title   string `json:"title" validate:"required"`
	Body    string `json:"body" validate:"required"`
	Data    string `json:"data,omitempty"`
}

type ListNotificationsRequest struct {
	Offset     int  `json:"offset"`
	Limit      int  `json:"limit"`
	UnreadOnly bool `json:"unread_only"`
}

type ListNotificationsResponse struct {
	Notifications []*NotificationResponse `json:"notifications"`
	Offset        int                     `json:"offset"`
	Limit         int                     `json:"limit"`
	Total         int                     `json:"total"`
}

type NotificationResponse struct {
	ID      string     `json:"id"`
	UserID  string     `json:"user_id"`
	Type    string     `json:"type"`
	Channel string     `json:"channel"`
	Title   string     `json:"title"`
	Body    string     `json:"body"`
	Data    string     `json:"data,omitempty"`
	Read    bool       `json:"read"`
	SentAt  *time.Time `json:"sent_at,omitempty"`
	ReadAt  *time.Time `json:"read_at,omitempty"`
}

// GenerateSalesReportRequest Report Service DTOs
type GenerateSalesReportRequest struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type GetSalesReportRequest struct {
	Date string `json:"date"`
}

type LowStockRequest struct {
	Threshold int `json:"threshold"`
}

type LowStockResponse struct {
	Threshold int               `json:"threshold"`
	Products  []ProductLowStock `json:"products"`
	Count     int               `json:"count"`
}

type ProductLowStock struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	SKU          string `json:"sku"`
	CurrentStock int    `json:"current_stock"`
	MinThreshold int    `json:"min_threshold"`
}

type LowStockItem struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	ProductSKU   string `json:"product_sku"`
	CurrentStock int    `json:"current_stock"`
	MinStock     int    `json:"min_stock"`
}

type SalesReportResponse struct {
	Date              string                 `json:"date"`
	TotalSales        float64                `json:"total_sales"`
	TotalOrders       int                    `json:"total_orders"`
	CompletedOrders   int                    `json:"completed_orders"`
	CancelledOrders   int                    `json:"cancelled_orders"`
	AverageOrderValue float64                `json:"average_order_value"`
	OrdersByStatus    map[string]int         `json:"orders_by_status"`
	Report            map[string]interface{} `json:"report"`
}

type InventoryReportResponse struct {
	TotalProducts      int                    `json:"total_products"`
	ActiveProducts     int                    `json:"active_products"`
	LowStockProducts   int                    `json:"low_stock_products"`
	OutOfStockProducts int                    `json:"out_of_stock_products"`
	TotalStockValue    float64                `json:"total_stock_value"`
	TotalStockQuantity int                    `json:"total_stock_quantity"`
	ProductInventory   []ProductInventoryItem `json:"product_inventory"`
}

type UserActivityReportRequest struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type UserActivityReportResponse struct {
	ActiveUsers int `json:"active_users"`
	NewUsers    int `json:"new_users"`
}

// ProductInventoryItem represents inventory information for a product
type ProductInventoryItem struct {
	ProductID     string  `json:"product_id"`
	ProductName   string  `json:"product_name"`
	SKU           string  `json:"sku"`
	Price         float64 `json:"price"`
	StockQuantity int     `json:"stock_quantity"`
	StockValue    float64 `json:"stock_value"`
	Status        string  `json:"status"`
}

// TopProductsResponse represents the response for the top products report
type TopProductsResponse struct {
	TopProducts []*TopProductItem `json:"top_products"`
	Limit       int               `json:"limit"`
	Period      string            `json:"period"`
}

// TopProductItem represents a single product in the top products report
type TopProductItem struct {
	ProductID     string  `json:"product_id"`
	ProductName   string  `json:"product_name"`
	TotalQuantity int     `json:"total_quantity"`
	TotalRevenue  float64 `json:"total_revenue"`
	OrderCount    int     `json:"order_count"`
}

// OrderPipelineResult represents the result of the order processing pipeline
type OrderPipelineResult struct {
	Order          *OrderResponse   `json:"order"`
	PaymentResult  *PaymentResponse `json:"payment,omitempty"`
	InventoryItems []InventoryItem  `json:"inventory_items,omitempty"`
	Notifications  []string         `json:"notifications,omitempty"`
	ProcessingTime time.Duration    `json:"processing_time"`
	Errors         []string         `json:"errors,omitempty"`
	Status         PipelineStatus   `json:"status"`
}

// PipelineStatus represents the status of pipeline execution
type PipelineStatus string

const (
	PipelineStatusPending    PipelineStatus = "pending"
	PipelineStatusProcessing PipelineStatus = "processing"
	PipelineStatusCompleted  PipelineStatus = "completed"
	PipelineStatusFailed     PipelineStatus = "failed"
	PipelineStatusPartial    PipelineStatus = "partial"
)
