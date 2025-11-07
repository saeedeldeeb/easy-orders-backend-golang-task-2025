package integration_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/repository"
	"easy-orders-backend/internal/services"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/errors"
	"easy-orders-backend/pkg/logger"
	"easy-orders-backend/tests/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// OrderConcurrencyTestSuite tests concurrent order processing scenarios
type OrderConcurrencyTestSuite struct {
	suite.Suite
	db               *database.DB
	ctx              context.Context
	orderService     services.OrderService
	inventoryService services.InventoryService
	orderRepo        repository.OrderRepository
	productRepo      repository.ProductRepository
	inventoryRepo    repository.InventoryRepository
	userRepo         repository.UserRepository
	log              *logger.Logger
}

// SetupSuite runs once before all tests
func (suite *OrderConcurrencyTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Initialize logger
	suite.log = testutil.NewTestLogger()

	// Connect to test database
	db, err := testutil.SetupTestDatabase()
	require.NoError(suite.T(), err)
	suite.db = db

	// Run migrations
	err = testutil.RunMigrations(db)
	require.NoError(suite.T(), err)
}

// SetupTest runs before each test
func (suite *OrderConcurrencyTestSuite) SetupTest() {
	// Clean up database before each test
	testutil.CleanDatabase(suite.db)

	// Initialize repositories
	suite.orderRepo = repository.NewOrderRepository(suite.db, suite.log)
	suite.productRepo = repository.NewProductRepository(suite.db, suite.log)
	suite.inventoryRepo = repository.NewInventoryRepository(suite.db, suite.log)
	suite.userRepo = repository.NewUserRepository(suite.db, suite.log)
	orderItemRepo := repository.NewOrderItemRepository(suite.db, suite.log)

	// Initialize services
	suite.inventoryService = services.NewInventoryService(
		suite.inventoryRepo,
		suite.productRepo,
		suite.log,
	)

	suite.orderService = services.NewOrderService(
		suite.db,
		suite.orderRepo,
		orderItemRepo,
		suite.productRepo,
		suite.inventoryRepo,
		suite.userRepo,
		suite.inventoryService,
		suite.log,
	)
}

// TearDownSuite runs once after all tests
func (suite *OrderConcurrencyTestSuite) TearDownSuite() {
	if suite.db != nil {
		testutil.TeardownTestDatabase(suite.db)
	}
}

// TestConcurrentOrdersForSameProduct tests multiple concurrent orders for the same product
// This ensures no overselling occurs when multiple orders are placed simultaneously
func (suite *OrderConcurrencyTestSuite) TestConcurrentOrdersForSameProduct() {
	// Create test user
	user := testutil.CreateTestUser(nil)
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Create product with limited stock
	stockQuantity := 100
	product := testutil.CreateTestProduct(func(p *models.Product) {
		p.IsActive = true
		p.Price = 50.00
	})

	inventory := testutil.CreateTestInventory(product.ID, func(i *models.Inventory) {
		i.Quantity = stockQuantity
		i.Available = stockQuantity
		i.Reserved = 0
	})

	err = suite.productRepo.CreateWithInventory(suite.ctx, product, inventory)
	require.NoError(suite.T(), err)

	// Number of concurrent orders
	numOrders := 20
	itemsPerOrder := 10 // Each order wants 10 items

	// Expected: Only 10 orders should succeed (100 items / 10 items per order)
	expectedSuccessfulOrders := stockQuantity / itemsPerOrder

	var wg sync.WaitGroup
	var successCount int32
	var failCount int32
	var mu sync.Mutex
	successfulOrderIDs := make([]string, 0)

	// Launch concurrent orders
	for i := 0; i < numOrders; i++ {
		wg.Add(1)
		go func(orderNum int) {
			defer wg.Done()

			req := services.CreateOrderRequest{
				UserID: user.ID,
				Items: []services.OrderItem{
					{
						ProductID: product.ID,
						Quantity:  itemsPerOrder,
					},
				},
				Notes: fmt.Sprintf("Concurrent order #%d", orderNum),
			}

			// Add slight random delay to increase concurrency conflicts
			time.Sleep(time.Millisecond * time.Duration(orderNum%5))

			response, err := suite.orderService.CreateOrder(suite.ctx, req)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
				mu.Lock()
				successfulOrderIDs = append(successfulOrderIDs, response.ID)
				mu.Unlock()
			} else {
				atomic.AddInt32(&failCount, 1)
				// Should fail with insufficient stock or reservation conflict
				assert.True(suite.T(),
					errors.IsErrorType(err, errors.ErrorTypeInsufficientStock) ||
						errors.IsErrorType(err, errors.ErrorTypeStockReservationConflict),
					"Error should be insufficient stock or reservation conflict, got: %v", err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify results
	suite.T().Logf("Successful orders: %d, Failed orders: %d", successCount, failCount)

	// Exactly 10 orders should succeed
	assert.Equal(suite.T(), int32(expectedSuccessfulOrders), successCount,
		"Expected exactly %d successful orders", expectedSuccessfulOrders)
	assert.Equal(suite.T(), int32(numOrders-expectedSuccessfulOrders), failCount,
		"Expected exactly %d failed orders", numOrders-expectedSuccessfulOrders)

	// Verify inventory state
	finalInventory, err := suite.inventoryRepo.GetByProductID(suite.ctx, product.ID)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), finalInventory)

	// All stock should be reserved (100 items reserved by 10 successful orders)
	assert.Equal(suite.T(), stockQuantity, finalInventory.Reserved,
		"All stock should be reserved")
	assert.Equal(suite.T(), 0, finalInventory.Available,
		"No stock should be available")
	assert.Equal(suite.T(), stockQuantity, finalInventory.Quantity,
		"Total quantity should remain unchanged")

	// Verify all successful orders exist in database
	for _, orderID := range successfulOrderIDs {
		order, err := suite.orderRepo.GetByID(suite.ctx, orderID)
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), order)
		assert.Equal(suite.T(), models.OrderStatusPending, order.Status)
	}
}

// TestConcurrentOrdersMultipleProducts tests concurrent orders for multiple products
func (suite *OrderConcurrencyTestSuite) TestConcurrentOrdersMultipleProducts() {
	// Create test user
	user := testutil.CreateTestUser(nil)
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Create two products with limited stock
	product1 := testutil.CreateTestProduct(func(p *models.Product) {
		p.IsActive = true
		p.Price = 30.00
	})
	inventory1 := testutil.CreateTestInventory(product1.ID, func(i *models.Inventory) {
		i.Quantity = 50
		i.Available = 50
	})
	err = suite.productRepo.CreateWithInventory(suite.ctx, product1, inventory1)
	require.NoError(suite.T(), err)

	product2 := testutil.CreateTestProduct(func(p *models.Product) {
		p.IsActive = true
		p.Price = 40.00
	})
	inventory2 := testutil.CreateTestInventory(product2.ID, func(i *models.Inventory) {
		i.Quantity = 50
		i.Available = 50
	})
	err = suite.productRepo.CreateWithInventory(suite.ctx, product2, inventory2)
	require.NoError(suite.T(), err)

	// Launch concurrent orders with both products
	numOrders := 15
	var wg sync.WaitGroup
	var successCount int32

	for i := 0; i < numOrders; i++ {
		wg.Add(1)
		go func(orderNum int) {
			defer wg.Done()

			req := services.CreateOrderRequest{
				UserID: user.ID,
				Items: []services.OrderItem{
					{ProductID: product1.ID, Quantity: 5},
					{ProductID: product2.ID, Quantity: 5},
				},
				Notes: fmt.Sprintf("Multi-product order #%d", orderNum),
			}

			_, err := suite.orderService.CreateOrder(suite.ctx, req)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// Expected: 10 orders should succeed (50 items / 5 items per order for each product)
	suite.T().Logf("Successful multi-product orders: %d out of %d", successCount, numOrders)
	assert.Equal(suite.T(), int32(10), successCount, "Expected 10 successful orders")

	// Verify both inventories
	inv1, _ := suite.inventoryRepo.GetByProductID(suite.ctx, product1.ID)
	inv2, _ := suite.inventoryRepo.GetByProductID(suite.ctx, product2.ID)

	assert.Equal(suite.T(), 50, inv1.Reserved, "Product 1: All stock reserved")
	assert.Equal(suite.T(), 0, inv1.Available, "Product 1: No stock available")
	assert.Equal(suite.T(), 50, inv2.Reserved, "Product 2: All stock reserved")
	assert.Equal(suite.T(), 0, inv2.Available, "Product 2: No stock available")
}

// TestOptimisticLockingConflict tests that optimistic locking prevents version conflicts
func (suite *OrderConcurrencyTestSuite) TestOptimisticLockingConflict() {
	// Create test user
	user := testutil.CreateTestUser(nil)
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Create product with very limited stock
	product := testutil.CreateTestProduct(func(p *models.Product) {
		p.IsActive = true
		p.Price = 100.00
	})
	inventory := testutil.CreateTestInventory(product.ID, func(i *models.Inventory) {
		i.Quantity = 10
		i.Available = 10
	})
	err = suite.productRepo.CreateWithInventory(suite.ctx, product, inventory)
	require.NoError(suite.T(), err)

	// Launch many concurrent small orders
	numOrders := 100
	var wg sync.WaitGroup
	var successCount int32
	var conflictCount int32

	for i := 0; i < numOrders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := services.CreateOrderRequest{
				UserID: user.ID,
				Items: []services.OrderItem{
					{ProductID: product.ID, Quantity: 1},
				},
			}

			_, err := suite.orderService.CreateOrder(suite.ctx, req)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else if errors.IsErrorType(err, errors.ErrorTypeStockReservationConflict) {
				atomic.AddInt32(&conflictCount, 1)
			}
		}()
	}

	wg.Wait()

	suite.T().Logf("Success: %d, Conflicts: %d, Other failures: %d",
		successCount, conflictCount, int32(numOrders)-successCount-conflictCount)

	// Exactly 10 orders should succeed (matching available inventory)
	assert.Equal(suite.T(), int32(10), successCount, "Expected exactly 10 successful orders")

	// Verify final inventory state
	inv, _ := suite.inventoryRepo.GetByProductID(suite.ctx, product.ID)
	assert.Equal(suite.T(), 10, inv.Reserved)
	assert.Equal(suite.T(), 0, inv.Available)
}

// TestTransactionRollback tests that failed orders rollback properly
func (suite *OrderConcurrencyTestSuite) TestTransactionRollback() {
	// Create test user
	user := testutil.CreateTestUser(nil)
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Create two products - one with enough stock, one without
	product1 := testutil.CreateTestProduct(func(p *models.Product) {
		p.IsActive = true
		p.Price = 50.00
	})
	inventory1 := testutil.CreateTestInventory(product1.ID, func(i *models.Inventory) {
		i.Quantity = 100
		i.Available = 100
	})
	err = suite.productRepo.CreateWithInventory(suite.ctx, product1, inventory1)
	require.NoError(suite.T(), err)

	product2 := testutil.CreateTestProduct(func(p *models.Product) {
		p.IsActive = true
		p.Price = 50.00
	})
	inventory2 := testutil.CreateTestInventory(product2.ID, func(i *models.Inventory) {
		i.Quantity = 5
		i.Available = 5
	})
	err = suite.productRepo.CreateWithInventory(suite.ctx, product2, inventory2)
	require.NoError(suite.T(), err)

	// Try to order more than available for product2
	req := services.CreateOrderRequest{
		UserID: user.ID,
		Items: []services.OrderItem{
			{ProductID: product1.ID, Quantity: 10},
			{ProductID: product2.ID, Quantity: 10}, // This should fail
		},
	}

	_, err = suite.orderService.CreateOrder(suite.ctx, req)
	assert.Error(suite.T(), err, "Order should fail due to insufficient stock")
	assert.True(suite.T(), errors.IsErrorType(err, errors.ErrorTypeInsufficientStock))

	// Verify no inventory was reserved for product1 (transaction rolled back)
	inv1, _ := suite.inventoryRepo.GetByProductID(suite.ctx, product1.ID)
	assert.Equal(suite.T(), 0, inv1.Reserved, "Product 1 reservation should be rolled back")
	assert.Equal(suite.T(), 100, inv1.Available, "Product 1 should still have all stock available")

	// Verify no order was created
	orders, _ := suite.orderRepo.List(suite.ctx, 0, 10)
	assert.Empty(suite.T(), orders, "No orders should be created")
}

// TestConcurrentOrdersWithDifferentQuantities tests various order sizes concurrently
func (suite *OrderConcurrencyTestSuite) TestConcurrentOrdersWithDifferentQuantities() {
	// Create test user
	user := testutil.CreateTestUser(nil)
	err := suite.userRepo.Create(suite.ctx, user)
	require.NoError(suite.T(), err)

	// Create product with stock
	product := testutil.CreateTestProduct(func(p *models.Product) {
		p.IsActive = true
		p.Price = 25.00
	})
	inventory := testutil.CreateTestInventory(product.ID, func(i *models.Inventory) {
		i.Quantity = 100
		i.Available = 100
	})
	err = suite.productRepo.CreateWithInventory(suite.ctx, product, inventory)
	require.NoError(suite.T(), err)

	// Different order quantities
	orderQuantities := []int{1, 5, 10, 15, 20}
	var wg sync.WaitGroup
	var totalReserved int32

	for _, qty := range orderQuantities {
		for i := 0; i < 5; i++ { // 5 orders of each quantity
			wg.Add(1)
			go func(quantity int) {
				defer wg.Done()

				req := services.CreateOrderRequest{
					UserID: user.ID,
					Items: []services.OrderItem{
						{ProductID: product.ID, Quantity: quantity},
					},
				}

				_, err := suite.orderService.CreateOrder(suite.ctx, req)
				if err == nil {
					atomic.AddInt32(&totalReserved, int32(quantity))
				}
			}(qty)
		}
	}

	wg.Wait()

	// Verify total reserved doesn't exceed available stock
	inv, _ := suite.inventoryRepo.GetByProductID(suite.ctx, product.ID)
	assert.LessOrEqual(suite.T(), inv.Reserved, 100, "Reserved should not exceed initial stock")
	assert.Equal(suite.T(), int(totalReserved), inv.Reserved, "Reserved matches sum of successful orders")
	assert.Equal(suite.T(), 100-inv.Reserved, inv.Available, "Available should be correct")
}

// TestOrderServiceTestSuite runs the test suite
func TestOrderConcurrencyTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(OrderConcurrencyTestSuite))
}
