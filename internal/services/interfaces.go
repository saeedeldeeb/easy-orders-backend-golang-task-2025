package services

import (
	"context"

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

// Request/Response structs
type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
}

type UpdateUserRequest struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty" validate:"omitempty,email"`
}

type ListUsersRequest struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type ListUsersResponse struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type CreateProductRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	CategoryID  string  `json:"category_id"`
}

type UpdateProductRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Price       *float64 `json:"price,omitempty" validate:"omitempty,gt=0"`
	CategoryID  *string  `json:"category_id,omitempty"`
}

type ListProductsRequest struct {
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
	CategoryID string `json:"category_id,omitempty"`
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
	CategoryID  string  `json:"category_id"`
}

type ListProductsResponse struct {
	Products []ProductResponse `json:"products"`
	Total    int               `json:"total"`
}

type CreateOrderRequest struct {
	UserID string      `json:"user_id" validate:"required"`
	Items  []OrderItem `json:"items" validate:"required,dive"`
}

type OrderItem struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,gt=0"`
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
	Orders []OrderResponse `json:"orders"`
	Total  int             `json:"total"`
}

type InventoryItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type ProcessPaymentRequest struct {
	OrderID     string  `json:"order_id" validate:"required"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	PaymentType string  `json:"payment_type" validate:"required"`
}

type PaymentResponse struct {
	ID      string               `json:"id"`
	OrderID string               `json:"order_id"`
	Amount  float64              `json:"amount"`
	Status  models.PaymentStatus `json:"status"`
}

type SendNotificationRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Title  string `json:"title" validate:"required"`
	Body   string `json:"body" validate:"required"`
}

type ListNotificationsRequest struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type ListNotificationsResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Total         int                    `json:"total"`
}

type NotificationResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Read  bool   `json:"read"`
}

type LowStockResponse struct {
	Items []LowStockItem `json:"items"`
}

type LowStockItem struct {
	ProductID string `json:"product_id"`
	Current   int    `json:"current"`
	Threshold int    `json:"threshold"`
}

type SalesReportResponse struct {
	Date         string  `json:"date"`
	TotalOrders  int     `json:"total_orders"`
	TotalRevenue float64 `json:"total_revenue"`
}

type InventoryReportResponse struct {
	TotalProducts int `json:"total_products"`
	TotalStock    int `json:"total_stock"`
}

type UserActivityReportRequest struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type UserActivityReportResponse struct {
	ActiveUsers int `json:"active_users"`
	NewUsers    int `json:"new_users"`
}
