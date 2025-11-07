package services_test

import (
	"context"
	"errors"
	"testing"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/logger"
	"easy-orders-backend/tests/mocks"
	"easy-orders-backend/tests/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// OrderServiceTestSuite defines the test suite for OrderService
type OrderServiceTestSuite struct {
	suite.Suite
	orderService     services.OrderService
	inventoryService services.InventoryService
	orderRepo        *mocks.MockOrderRepository
	orderItemRepo    *mocks.MockOrderItemRepository
	productRepo      *mocks.MockProductRepository
	inventoryRepo    *mocks.MockInventoryRepository
	userRepo         *mocks.MockUserRepository
	logger           *logger.Logger
	ctx              context.Context
}

// SetupTest runs before each test in the suite
func (suite *OrderServiceTestSuite) SetupTest() {
	suite.orderRepo = new(mocks.MockOrderRepository)
	suite.orderItemRepo = new(mocks.MockOrderItemRepository)
	suite.productRepo = new(mocks.MockProductRepository)
	suite.inventoryRepo = new(mocks.MockInventoryRepository)
	suite.userRepo = new(mocks.MockUserRepository)
	suite.logger = &logger.Logger{SugaredLogger: mocks.NewNoOpLogger()}
	suite.ctx = context.Background()

	// Create inventory service
	suite.inventoryService = services.NewInventoryService(
		suite.inventoryRepo,
		suite.productRepo,
		suite.logger,
	)

	// Note: For unit tests with mocks, we pass nil for DB since we're mocking repositories
	// In real scenarios, the transaction logic won't be tested with mocks
	suite.orderService = services.NewOrderService(
		nil, // DB not needed for unit tests with mocks
		suite.orderRepo,
		suite.orderItemRepo,
		suite.productRepo,
		suite.inventoryRepo,
		suite.userRepo,
		suite.inventoryService,
		suite.logger,
	)
}

// TearDownTest runs after each test in the suite
func (suite *OrderServiceTestSuite) TearDownTest() {
	suite.orderRepo.AssertExpectations(suite.T())
	suite.orderItemRepo.AssertExpectations(suite.T())
	suite.productRepo.AssertExpectations(suite.T())
	suite.inventoryRepo.AssertExpectations(suite.T())
	suite.userRepo.AssertExpectations(suite.T())
}

// Test CreateOrder - Happy Path
func (suite *OrderServiceTestSuite) TestCreateOrder_Success() {
	userID := "user-id-123"
	productID1 := "product-id-1"
	productID2 := "product-id-2"

	user := testutil.CreateTestUser(func(u *models.User) {
		u.ID = userID
	})

	product1 := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = productID1
		p.Price = 50.00
		p.IsActive = true
	})

	product2 := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = productID2
		p.Price = 30.00
		p.IsActive = true
	})

	inventory1 := testutil.CreateTestInventory(productID1, func(i *models.Inventory) {
		i.Available = 100
	})

	inventory2 := testutil.CreateTestInventory(productID2, func(i *models.Inventory) {
		i.Available = 50
	})

	req := services.CreateOrderRequest{
		UserID: userID,
		Items: []services.OrderItem{
			{ProductID: productID1, Quantity: 2},
			{ProductID: productID2, Quantity: 1},
		},
		Notes: "Test order",
	}

	// Mock expectations
	suite.userRepo.On("GetByID", suite.ctx, userID).Return(user, nil)
	suite.productRepo.On("GetByID", suite.ctx, productID1).Return(product1, nil)
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID1).Return(inventory1, nil)
	suite.productRepo.On("GetByID", suite.ctx, productID2).Return(product2, nil)
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID2).Return(inventory2, nil)
	suite.orderRepo.On("Create", suite.ctx, mock.AnythingOfType("*models.Order")).
		Run(func(args mock.Arguments) {
			order := args.Get(1).(*models.Order)
			order.ID = "order-id-456"
		}).
		Return(nil)
	suite.orderItemRepo.On("CreateBatch", suite.ctx, mock.AnythingOfType("[]*models.OrderItem")).Return(nil)

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), userID, response.UserID)
	assert.Equal(suite.T(), models.OrderStatusPending, response.Status)
	assert.Equal(suite.T(), 2, len(response.Items))
	assert.Equal(suite.T(), 130.00, response.Total) // (50*2) + (30*1)
}

// Test CreateOrder - Validation Error: User ID Required
func (suite *OrderServiceTestSuite) TestCreateOrder_ValidationError_UserIDRequired() {
	req := services.CreateOrderRequest{
		UserID: "",
		Items: []services.OrderItem{
			{ProductID: "product-id", Quantity: 1},
		},
	}

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user ID is required")
}

// Test CreateOrder - Validation Error: No Items
func (suite *OrderServiceTestSuite) TestCreateOrder_ValidationError_NoItems() {
	req := services.CreateOrderRequest{
		UserID: "user-id-123",
		Items:  []services.OrderItem{},
	}

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "at least one item")
}

// Test CreateOrder - User Not Found
func (suite *OrderServiceTestSuite) TestCreateOrder_UserNotFound() {
	userID := "non-existent-user"
	req := services.CreateOrderRequest{
		UserID: userID,
		Items: []services.OrderItem{
			{ProductID: "product-id", Quantity: 1},
		},
	}

	// Mock expectations
	suite.userRepo.On("GetByID", suite.ctx, userID).Return(nil, nil)

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "user not found")
}

// Test CreateOrder - Repository Error on GetUser
func (suite *OrderServiceTestSuite) TestCreateOrder_RepositoryError_GetUser() {
	userID := "user-id-123"
	req := services.CreateOrderRequest{
		UserID: userID,
		Items: []services.OrderItem{
			{ProductID: "product-id", Quantity: 1},
		},
	}

	// Mock expectations
	suite.userRepo.On("GetByID", suite.ctx, userID).Return(nil, errors.New("database error"))

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test CreateOrder - Validation Error: Product ID Required
func (suite *OrderServiceTestSuite) TestCreateOrder_ValidationError_ProductIDRequired() {
	userID := "user-id-123"
	user := testutil.CreateTestUser(func(u *models.User) {
		u.ID = userID
	})

	req := services.CreateOrderRequest{
		UserID: userID,
		Items: []services.OrderItem{
			{ProductID: "", Quantity: 1},
		},
	}

	// Mock expectations
	suite.userRepo.On("GetByID", suite.ctx, userID).Return(user, nil)

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "product ID is required")
}

// Test CreateOrder - Validation Error: Invalid Quantity
func (suite *OrderServiceTestSuite) TestCreateOrder_ValidationError_InvalidQuantity() {
	userID := "user-id-123"
	user := testutil.CreateTestUser(func(u *models.User) {
		u.ID = userID
	})

	req := services.CreateOrderRequest{
		UserID: userID,
		Items: []services.OrderItem{
			{ProductID: "product-id", Quantity: 0},
		},
	}

	// Mock expectations
	suite.userRepo.On("GetByID", suite.ctx, userID).Return(user, nil)

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "quantity must be greater than 0")
}

// Test CreateOrder - Product Not Found
func (suite *OrderServiceTestSuite) TestCreateOrder_ProductNotFound() {
	userID := "user-id-123"
	productID := "non-existent-product"

	user := testutil.CreateTestUser(func(u *models.User) {
		u.ID = userID
	})

	req := services.CreateOrderRequest{
		UserID: userID,
		Items: []services.OrderItem{
			{ProductID: productID, Quantity: 1},
		},
	}

	// Mock expectations
	suite.userRepo.On("GetByID", suite.ctx, userID).Return(user, nil)
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(nil, nil)

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// Test CreateOrder - Product Inactive
func (suite *OrderServiceTestSuite) TestCreateOrder_ProductInactive() {
	userID := "user-id-123"
	productID := "product-id-456"

	user := testutil.CreateTestUser(func(u *models.User) {
		u.ID = userID
	})

	product := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = productID
		p.IsActive = false
	})

	req := services.CreateOrderRequest{
		UserID: userID,
		Items: []services.OrderItem{
			{ProductID: productID, Quantity: 1},
		},
	}

	// Mock expectations
	suite.userRepo.On("GetByID", suite.ctx, userID).Return(user, nil)
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(product, nil)

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "not available")
}

// Test CreateOrder - Insufficient Stock
func (suite *OrderServiceTestSuite) TestCreateOrder_InsufficientStock() {
	userID := "user-id-123"
	productID := "product-id-456"

	user := testutil.CreateTestUser(func(u *models.User) {
		u.ID = userID
	})

	product := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = productID
		p.IsActive = true
	})

	inventory := testutil.CreateTestInventory(productID, func(i *models.Inventory) {
		i.Available = 5
	})

	req := services.CreateOrderRequest{
		UserID: userID,
		Items: []services.OrderItem{
			{ProductID: productID, Quantity: 10},
		},
	}

	// Mock expectations
	suite.userRepo.On("GetByID", suite.ctx, userID).Return(user, nil)
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(product, nil)
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(inventory, nil)

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "insufficient stock")
}

// Test CreateOrder - No Inventory
func (suite *OrderServiceTestSuite) TestCreateOrder_NoInventory() {
	userID := "user-id-123"
	productID := "product-id-456"

	user := testutil.CreateTestUser(func(u *models.User) {
		u.ID = userID
	})

	product := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = productID
		p.IsActive = true
	})

	req := services.CreateOrderRequest{
		UserID: userID,
		Items: []services.OrderItem{
			{ProductID: productID, Quantity: 1},
		},
	}

	// Mock expectations
	suite.userRepo.On("GetByID", suite.ctx, userID).Return(user, nil)
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(product, nil)
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(nil, nil)

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "insufficient stock")
}

// Test CreateOrder - Repository Error on Create
func (suite *OrderServiceTestSuite) TestCreateOrder_RepositoryError_Create() {
	userID := "user-id-123"
	productID := "product-id-456"

	user := testutil.CreateTestUser(func(u *models.User) {
		u.ID = userID
	})

	product := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = productID
		p.Price = 50.00
		p.IsActive = true
	})

	inventory := testutil.CreateTestInventory(productID, func(i *models.Inventory) {
		i.Available = 100
	})

	req := services.CreateOrderRequest{
		UserID: userID,
		Items: []services.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	// Mock expectations
	suite.userRepo.On("GetByID", suite.ctx, userID).Return(user, nil)
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(product, nil)
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(inventory, nil)
	suite.orderRepo.On("Create", suite.ctx, mock.AnythingOfType("*models.Order")).Return(errors.New("database error"))

	// Execute
	response, err := suite.orderService.CreateOrder(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test GetOrder - Happy Path
func (suite *OrderServiceTestSuite) TestGetOrder_Success() {
	orderID := "order-id-123"
	userID := "user-id-456"

	order := testutil.CreateTestOrder(userID, func(o *models.Order) {
		o.ID = orderID
		o.Status = models.OrderStatusPending
		o.TotalAmount = 100.00
		o.Items = []models.OrderItem{
			*testutil.CreateTestOrderItem(orderID, "product-id-1", func(i *models.OrderItem) {
				i.Quantity = 2
				i.UnitPrice = 50.00
			}),
		}
	})

	// Mock expectations
	suite.orderRepo.On("GetByIDWithItems", suite.ctx, orderID).Return(order, nil)

	// Execute
	response, err := suite.orderService.GetOrder(suite.ctx, orderID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), orderID, response.ID)
	assert.Equal(suite.T(), userID, response.UserID)
	assert.Equal(suite.T(), 100.00, response.Total)
	assert.Equal(suite.T(), 1, len(response.Items))
}

// Test GetOrder - Validation Error: ID Required
func (suite *OrderServiceTestSuite) TestGetOrder_ValidationError_IDRequired() {
	// Execute
	response, err := suite.orderService.GetOrder(suite.ctx, "")

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "order ID is required")
}

// Test GetOrder - Order Not Found
func (suite *OrderServiceTestSuite) TestGetOrder_NotFound() {
	orderID := "non-existent-order"

	// Mock expectations
	suite.orderRepo.On("GetByIDWithItems", suite.ctx, orderID).Return(nil, nil)

	// Execute
	response, err := suite.orderService.GetOrder(suite.ctx, orderID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "order not found")
}

// Test GetOrder - Repository Error
func (suite *OrderServiceTestSuite) TestGetOrder_RepositoryError() {
	orderID := "order-id-789"

	// Mock expectations
	suite.orderRepo.On("GetByIDWithItems", suite.ctx, orderID).Return(nil, errors.New("database error"))

	// Execute
	response, err := suite.orderService.GetOrder(suite.ctx, orderID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test UpdateOrderStatus - Happy Path
func (suite *OrderServiceTestSuite) TestUpdateOrderStatus_Success() {
	orderID := "order-id-123"
	userID := "user-id-456"

	order := testutil.CreateTestOrder(userID, func(o *models.Order) {
		o.ID = orderID
		o.Status = models.OrderStatusPending
	})

	updatedOrder := testutil.CreateTestOrder(userID, func(o *models.Order) {
		o.ID = orderID
		o.Status = models.OrderStatusConfirmed
		o.Items = []models.OrderItem{}
	})

	// Mock expectations
	suite.orderRepo.On("GetByID", suite.ctx, orderID).Return(order, nil)
	suite.orderRepo.On("UpdateStatus", suite.ctx, orderID, models.OrderStatusConfirmed).Return(nil)
	suite.orderRepo.On("GetByIDWithItems", suite.ctx, orderID).Return(updatedOrder, nil)

	// Execute
	response, err := suite.orderService.UpdateOrderStatus(suite.ctx, orderID, models.OrderStatusConfirmed)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), models.OrderStatusConfirmed, response.Status)
}

// Test UpdateOrderStatus - Validation Error: ID Required
func (suite *OrderServiceTestSuite) TestUpdateOrderStatus_ValidationError_IDRequired() {
	// Execute
	response, err := suite.orderService.UpdateOrderStatus(suite.ctx, "", models.OrderStatusConfirmed)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "order ID is required")
}

// Test UpdateOrderStatus - Order Not Found
func (suite *OrderServiceTestSuite) TestUpdateOrderStatus_NotFound() {
	orderID := "non-existent-order"

	// Mock expectations
	suite.orderRepo.On("GetByID", suite.ctx, orderID).Return(nil, nil)

	// Execute
	response, err := suite.orderService.UpdateOrderStatus(suite.ctx, orderID, models.OrderStatusConfirmed)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "order not found")
}

// Test UpdateOrderStatus - Invalid Transition
func (suite *OrderServiceTestSuite) TestUpdateOrderStatus_InvalidTransition() {
	orderID := "order-id-123"
	userID := "user-id-456"

	order := testutil.CreateTestOrder(userID, func(o *models.Order) {
		o.ID = orderID
		o.Status = models.OrderStatusDelivered
	})

	// Mock expectations
	suite.orderRepo.On("GetByID", suite.ctx, orderID).Return(order, nil)

	// Execute
	response, err := suite.orderService.UpdateOrderStatus(suite.ctx, orderID, models.OrderStatusPending)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "cannot transition")
}

// Test UpdateOrderStatus - Repository Error on UpdateStatus
func (suite *OrderServiceTestSuite) TestUpdateOrderStatus_RepositoryError() {
	orderID := "order-id-123"
	userID := "user-id-456"

	order := testutil.CreateTestOrder(userID, func(o *models.Order) {
		o.ID = orderID
		o.Status = models.OrderStatusPending
	})

	// Mock expectations
	suite.orderRepo.On("GetByID", suite.ctx, orderID).Return(order, nil)
	suite.orderRepo.On("UpdateStatus", suite.ctx, orderID, models.OrderStatusConfirmed).Return(errors.New("database error"))

	// Execute
	response, err := suite.orderService.UpdateOrderStatus(suite.ctx, orderID, models.OrderStatusConfirmed)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test CancelOrder - Happy Path
func (suite *OrderServiceTestSuite) TestCancelOrder_Success() {
	orderID := "order-id-123"
	userID := "user-id-456"

	order := testutil.CreateTestOrder(userID, func(o *models.Order) {
		o.ID = orderID
		o.Status = models.OrderStatusPending
	})

	// Mock expectations
	suite.orderRepo.On("GetByID", suite.ctx, orderID).Return(order, nil)
	suite.orderRepo.On("UpdateStatus", suite.ctx, orderID, models.OrderStatusCancelled).Return(nil)

	// Execute
	err := suite.orderService.CancelOrder(suite.ctx, orderID)

	// Assert
	assert.NoError(suite.T(), err)
}

// Test CancelOrder - Validation Error: ID Required
func (suite *OrderServiceTestSuite) TestCancelOrder_ValidationError_IDRequired() {
	// Execute
	err := suite.orderService.CancelOrder(suite.ctx, "")

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "order ID is required")
}

// Test CancelOrder - Order Not Found
func (suite *OrderServiceTestSuite) TestCancelOrder_NotFound() {
	orderID := "non-existent-order"

	// Mock expectations
	suite.orderRepo.On("GetByID", suite.ctx, orderID).Return(nil, nil)

	// Execute
	err := suite.orderService.CancelOrder(suite.ctx, orderID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "order not found")
}

// Test CancelOrder - Order Not Cancellable
func (suite *OrderServiceTestSuite) TestCancelOrder_NotCancellable() {
	orderID := "order-id-123"
	userID := "user-id-456"

	order := testutil.CreateTestOrder(userID, func(o *models.Order) {
		o.ID = orderID
		o.Status = models.OrderStatusDelivered // Not cancellable
	})

	// Mock expectations
	suite.orderRepo.On("GetByID", suite.ctx, orderID).Return(order, nil)

	// Execute
	err := suite.orderService.CancelOrder(suite.ctx, orderID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "cannot be cancelled")
}

// Test ListOrders - Happy Path
func (suite *OrderServiceTestSuite) TestListOrders_Success() {
	userID := "user-id-123"

	orders := []*models.Order{
		testutil.CreateTestOrder(userID, func(o *models.Order) {
			o.ID = "order-1"
			o.Items = []models.OrderItem{}
		}),
		testutil.CreateTestOrder(userID, func(o *models.Order) {
			o.ID = "order-2"
			o.Items = []models.OrderItem{}
		}),
	}

	req := services.ListOrdersRequest{
		Page:  1,
		Limit: 20,
	}

	// Mock expectations
	suite.orderRepo.On("List", suite.ctx, 0, 20).Return(orders, nil)
	suite.orderRepo.On("Count", suite.ctx).Return(int64(2), nil)

	// Execute
	response, err := suite.orderService.ListOrders(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 2, len(response.Orders))
	assert.Equal(suite.T(), 1, response.Page)
	assert.Equal(suite.T(), 20, response.Limit)
	assert.Equal(suite.T(), 2, response.Total)
}

// Test ListOrders - Filter By Status
func (suite *OrderServiceTestSuite) TestListOrders_FilterByStatus() {
	userID := "user-id-123"

	orders := []*models.Order{
		testutil.CreateTestOrder(userID, func(o *models.Order) {
			o.ID = "order-1"
			o.Status = models.OrderStatusPending
			o.Items = []models.OrderItem{}
		}),
	}

	req := services.ListOrdersRequest{
		Page:   1,
		Limit:  20,
		Status: models.OrderStatusPending,
	}

	// Mock expectations
	suite.orderRepo.On("ListByStatus", suite.ctx, models.OrderStatusPending, 0, 20).Return(orders, nil)
	suite.orderRepo.On("CountByStatus", suite.ctx, models.OrderStatusPending).Return(int64(1), nil)

	// Execute
	response, err := suite.orderService.ListOrders(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 1, len(response.Orders))
}

// Test ListOrders - Default Pagination
func (suite *OrderServiceTestSuite) TestListOrders_DefaultPagination() {
	orders := []*models.Order{}

	req := services.ListOrdersRequest{
		Page:  0, // Invalid, should default to 1
		Limit: 0, // Invalid, should default to 20
	}

	// Mock expectations
	suite.orderRepo.On("List", suite.ctx, 0, 20).Return(orders, nil)
	suite.orderRepo.On("Count", suite.ctx).Return(int64(0), nil)

	// Execute
	response, err := suite.orderService.ListOrders(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 1, response.Page)
	assert.Equal(suite.T(), 20, response.Limit)
}

// Test ListOrders - Repository Error on List
func (suite *OrderServiceTestSuite) TestListOrders_RepositoryError_List() {
	req := services.ListOrdersRequest{
		Page:  1,
		Limit: 20,
	}

	// Mock expectations
	suite.orderRepo.On("List", suite.ctx, 0, 20).Return(nil, errors.New("database error"))

	// Execute
	response, err := suite.orderService.ListOrders(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test ListOrders - Repository Error on Count
func (suite *OrderServiceTestSuite) TestListOrders_RepositoryError_Count() {
	orders := []*models.Order{}
	req := services.ListOrdersRequest{
		Page:  1,
		Limit: 20,
	}

	// Mock expectations
	suite.orderRepo.On("List", suite.ctx, 0, 20).Return(orders, nil)
	suite.orderRepo.On("Count", suite.ctx).Return(int64(0), errors.New("database error"))

	// Execute
	response, err := suite.orderService.ListOrders(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// TestOrderServiceTestSuite runs the test suite
func TestOrderServiceTestSuite(t *testing.T) {
	suite.Run(t, new(OrderServiceTestSuite))
}
