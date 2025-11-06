package testutil

import (
	"time"

	"easy-orders-backend/internal/models"

	"github.com/google/uuid"
)

// CreateTestUser creates a test user with default values
func CreateTestUser(overrides ...func(*models.User)) *models.User {
	user := &models.User{
		ID:        uuid.New().String(),
		Name:      "Test User",
		Email:     "test@example.com",
		Password:  "$2a$10$test.hash",
		Role:      models.UserRoleCustomer,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	for _, override := range overrides {
		override(user)
	}

	return user
}

// CreateTestProduct creates a test product with default values
func CreateTestProduct(overrides ...func(*models.Product)) *models.Product {
	product := &models.Product{
		ID:          uuid.New().String(),
		Name:        "Test Product",
		Description: "Test Description",
		SKU:         "TEST-SKU-" + uuid.New().String()[:8],
		Price:       99.99,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	for _, override := range overrides {
		override(product)
	}

	return product
}

// CreateTestInventory creates a test inventory with default values
func CreateTestInventory(productID string, overrides ...func(*models.Inventory)) *models.Inventory {
	inventory := &models.Inventory{
		ID:        1,
		ProductID: productID,
		Quantity:  100,
		Reserved:  0,
		Available: 100,
		MinStock:  10,
		MaxStock:  1000,
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	for _, override := range overrides {
		override(inventory)
	}

	return inventory
}

// CreateTestOrder creates a test order with default values
func CreateTestOrder(userID string, overrides ...func(*models.Order)) *models.Order {
	order := &models.Order{
		ID:          uuid.New().String(),
		UserID:      userID,
		Status:      models.OrderStatusPending,
		TotalAmount: 199.99,
		Currency:    "USD",
		Notes:       "Test order",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	for _, override := range overrides {
		override(order)
	}

	return order
}

// CreateTestOrderItem creates a test order item with default values
func CreateTestOrderItem(orderID, productID string, overrides ...func(*models.OrderItem)) *models.OrderItem {
	item := &models.OrderItem{
		ID:         uuid.New().String(),
		OrderID:    orderID,
		ProductID:  productID,
		Quantity:   2,
		UnitPrice:  99.99,
		TotalPrice: 199.98,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	for _, override := range overrides {
		override(item)
	}

	return item
}

// CreateTestPayment creates a test payment with default values
func CreateTestPayment(orderID string, overrides ...func(*models.Payment)) *models.Payment {
	payment := &models.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		Amount:        199.99,
		Method:        models.PaymentMethodCreditCard,
		Status:        models.PaymentStatusPending,
		TransactionID: "TXN-" + uuid.New().String()[:8],
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	for _, override := range overrides {
		override(payment)
	}

	return payment
}

// CreateTestNotification creates a test notification with default values
func CreateTestNotification(userID string, overrides ...func(*models.Notification)) *models.Notification {
	notification := &models.Notification{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      models.NotificationTypeOrderConfirmed,
		Title:     "Test Notification",
		Body:      "Test message body",
		Channel:   models.NotificationChannelInApp,
		Read:      false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	for _, override := range overrides {
		override(notification)
	}

	return notification
}

// Float64Ptr returns a pointer to a float64 value
func Float64Ptr(v float64) *float64 {
	return &v
}

// IntPtr returns a pointer to an int value
func IntPtr(v int) *int {
	return &v
}

// StringPtr returns a pointer to a string value
func StringPtr(v string) *string {
	return &v
}

// TimePtr returns a pointer to a time value
func TimePtr(v time.Time) *time.Time {
	return &v
}
