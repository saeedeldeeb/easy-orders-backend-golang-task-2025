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
	"github.com/stretchr/testify/suite"
)

// InventoryServiceTestSuite defines the test suite for InventoryService
type InventoryServiceTestSuite struct {
	suite.Suite
	inventoryService services.InventoryService
	inventoryRepo    *mocks.MockInventoryRepository
	productRepo      *mocks.MockProductRepository
	logger           *logger.Logger
	ctx              context.Context
}

// SetupTest runs before each test in the suite
func (suite *InventoryServiceTestSuite) SetupTest() {
	suite.inventoryRepo = new(mocks.MockInventoryRepository)
	suite.productRepo = new(mocks.MockProductRepository)
	suite.logger = &logger.Logger{SugaredLogger: mocks.NewNoOpLogger()}
	suite.ctx = context.Background()

	suite.inventoryService = services.NewInventoryService(
		suite.inventoryRepo,
		suite.productRepo,
		suite.logger,
	)
}

// TearDownTest runs after each test in the suite
func (suite *InventoryServiceTestSuite) TearDownTest() {
	suite.inventoryRepo.AssertExpectations(suite.T())
	suite.productRepo.AssertExpectations(suite.T())
}

// Test CheckAvailability - Happy Path (Sufficient Stock)
func (suite *InventoryServiceTestSuite) TestCheckAvailability_Success_SufficientStock() {
	productID := "product-id-123"
	inventory := testutil.CreateTestInventory(productID, func(i *models.Inventory) {
		i.Available = 100
		i.Reserved = 10
	})

	// Mock expectations
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(inventory, nil)

	// Execute
	available, err := suite.inventoryService.CheckAvailability(suite.ctx, productID, 50)

	// Assert
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), available)
}

// Test CheckAvailability - Insufficient Stock
func (suite *InventoryServiceTestSuite) TestCheckAvailability_InsufficientStock() {
	productID := "product-id-456"
	inventory := testutil.CreateTestInventory(productID, func(i *models.Inventory) {
		i.Available = 5
		i.Reserved = 0
	})

	// Mock expectations
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(inventory, nil)

	// Execute
	available, err := suite.inventoryService.CheckAvailability(suite.ctx, productID, 10)

	// Assert
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), available)
}

// Test CheckAvailability - No Inventory Found
func (suite *InventoryServiceTestSuite) TestCheckAvailability_NoInventory() {
	productID := "product-id-789"

	// Mock expectations
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(nil, nil)

	// Execute
	available, err := suite.inventoryService.CheckAvailability(suite.ctx, productID, 10)

	// Assert
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), available)
}

// Test CheckAvailability - Validation Error: Product ID Required
func (suite *InventoryServiceTestSuite) TestCheckAvailability_ValidationError_ProductIDRequired() {
	// Execute
	available, err := suite.inventoryService.CheckAvailability(suite.ctx, "", 10)

	// Assert
	assert.Error(suite.T(), err)
	assert.False(suite.T(), available)
	assert.Contains(suite.T(), err.Error(), "product ID is required")
}

// Test CheckAvailability - Validation Error: Invalid Quantity
func (suite *InventoryServiceTestSuite) TestCheckAvailability_ValidationError_InvalidQuantity() {
	// Execute
	available, err := suite.inventoryService.CheckAvailability(suite.ctx, "product-id", 0)

	// Assert
	assert.Error(suite.T(), err)
	assert.False(suite.T(), available)
	assert.Contains(suite.T(), err.Error(), "quantity must be greater than 0")
}

// Test CheckAvailability - Repository Error
func (suite *InventoryServiceTestSuite) TestCheckAvailability_RepositoryError() {
	productID := "product-id-999"

	// Mock expectations
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(nil, errors.New("database error"))

	// Execute
	available, err := suite.inventoryService.CheckAvailability(suite.ctx, productID, 10)

	// Assert
	assert.Error(suite.T(), err)
	assert.False(suite.T(), available)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test ReserveInventory - Happy Path
func (suite *InventoryServiceTestSuite) TestReserveInventory_Success() {
	productID1 := "product-id-1"
	productID2 := "product-id-2"

	inventory1 := testutil.CreateTestInventory(productID1, func(i *models.Inventory) {
		i.Available = 100
	})
	inventory2 := testutil.CreateTestInventory(productID2, func(i *models.Inventory) {
		i.Available = 50
	})

	items := []services.InventoryItem{
		{ProductID: productID1, Quantity: 10},
		{ProductID: productID2, Quantity: 5},
	}

	// Mock expectations
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID1).Return(inventory1, nil)
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID2).Return(inventory2, nil)
	suite.inventoryRepo.On("ReserveStock", suite.ctx, productID1, 10).Return(nil)
	suite.inventoryRepo.On("ReserveStock", suite.ctx, productID2, 5).Return(nil)

	// Execute
	err := suite.inventoryService.ReserveInventory(suite.ctx, items)

	// Assert
	assert.NoError(suite.T(), err)
}

// Test ReserveInventory - Validation Error: Empty Items
func (suite *InventoryServiceTestSuite) TestReserveInventory_ValidationError_EmptyItems() {
	// Execute
	err := suite.inventoryService.ReserveInventory(suite.ctx, []services.InventoryItem{})

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "no items to reserve")
}

// Test ReserveInventory - Validation Error: Product ID Required
func (suite *InventoryServiceTestSuite) TestReserveInventory_ValidationError_ProductIDRequired() {
	items := []services.InventoryItem{
		{ProductID: "", Quantity: 10},
	}

	// Execute
	err := suite.inventoryService.ReserveInventory(suite.ctx, items)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "product ID is required")
}

// Test ReserveInventory - Validation Error: Invalid Quantity
func (suite *InventoryServiceTestSuite) TestReserveInventory_ValidationError_InvalidQuantity() {
	items := []services.InventoryItem{
		{ProductID: "product-id", Quantity: 0},
	}

	// Execute
	err := suite.inventoryService.ReserveInventory(suite.ctx, items)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "quantity must be greater than 0")
}

// Test ReserveInventory - Insufficient Stock for One Item
func (suite *InventoryServiceTestSuite) TestReserveInventory_InsufficientStock() {
	productID := "product-id-1"
	inventory := testutil.CreateTestInventory(productID, func(i *models.Inventory) {
		i.Available = 5
	})

	items := []services.InventoryItem{
		{ProductID: productID, Quantity: 10},
	}

	// Mock expectations
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(inventory, nil)

	// Execute
	err := suite.inventoryService.ReserveInventory(suite.ctx, items)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "insufficient stock")
}

// Test ReserveInventory - Repository Error on CheckAvailability
func (suite *InventoryServiceTestSuite) TestReserveInventory_RepositoryError_CheckAvailability() {
	productID := "product-id-1"

	items := []services.InventoryItem{
		{ProductID: productID, Quantity: 10},
	}

	// Mock expectations
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(nil, errors.New("database error"))

	// Execute
	err := suite.inventoryService.ReserveInventory(suite.ctx, items)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test ReserveInventory - Repository Error on ReserveStock
func (suite *InventoryServiceTestSuite) TestReserveInventory_RepositoryError_ReserveStock() {
	productID := "product-id-1"
	inventory := testutil.CreateTestInventory(productID, func(i *models.Inventory) {
		i.Available = 100
	})

	items := []services.InventoryItem{
		{ProductID: productID, Quantity: 10},
	}

	// Mock expectations
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(inventory, nil)
	suite.inventoryRepo.On("ReserveStock", suite.ctx, productID, 10).Return(errors.New("database error"))

	// Execute
	err := suite.inventoryService.ReserveInventory(suite.ctx, items)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to reserve stock")
}

// Test ReleaseInventory - Happy Path
func (suite *InventoryServiceTestSuite) TestReleaseInventory_Success() {
	productID1 := "product-id-1"
	productID2 := "product-id-2"

	items := []services.InventoryItem{
		{ProductID: productID1, Quantity: 10},
		{ProductID: productID2, Quantity: 5},
	}

	// Mock expectations
	suite.inventoryRepo.On("ReleaseStock", suite.ctx, productID1, 10).Return(nil)
	suite.inventoryRepo.On("ReleaseStock", suite.ctx, productID2, 5).Return(nil)

	// Execute
	err := suite.inventoryService.ReleaseInventory(suite.ctx, items)

	// Assert
	assert.NoError(suite.T(), err)
}

// Test ReleaseInventory - Validation Error: Empty Items
func (suite *InventoryServiceTestSuite) TestReleaseInventory_ValidationError_EmptyItems() {
	// Execute
	err := suite.inventoryService.ReleaseInventory(suite.ctx, []services.InventoryItem{})

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "no items to release")
}

// Test ReleaseInventory - Validation Error: Product ID Required
func (suite *InventoryServiceTestSuite) TestReleaseInventory_ValidationError_ProductIDRequired() {
	items := []services.InventoryItem{
		{ProductID: "", Quantity: 10},
	}

	// Execute
	err := suite.inventoryService.ReleaseInventory(suite.ctx, items)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "product ID is required")
}

// Test ReleaseInventory - Validation Error: Invalid Quantity
func (suite *InventoryServiceTestSuite) TestReleaseInventory_ValidationError_InvalidQuantity() {
	items := []services.InventoryItem{
		{ProductID: "product-id", Quantity: -1},
	}

	// Execute
	err := suite.inventoryService.ReleaseInventory(suite.ctx, items)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "quantity must be greater than 0")
}

// Test ReleaseInventory - Repository Error
func (suite *InventoryServiceTestSuite) TestReleaseInventory_RepositoryError() {
	productID := "product-id-1"

	items := []services.InventoryItem{
		{ProductID: productID, Quantity: 10},
	}

	// Mock expectations
	suite.inventoryRepo.On("ReleaseStock", suite.ctx, productID, 10).Return(errors.New("database error"))

	// Execute
	err := suite.inventoryService.ReleaseInventory(suite.ctx, items)

	// Assert
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to release stock")
}

// Test GetLowStockAlert - Happy Path
func (suite *InventoryServiceTestSuite) TestGetLowStockAlert_Success() {
	product1 := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = "product-1"
		p.Name = "Low Stock Product 1"
		p.SKU = "SKU-001"
	})
	product2 := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = "product-2"
		p.Name = "Low Stock Product 2"
		p.SKU = "SKU-002"
	})

	lowStockItems := []*models.Inventory{
		testutil.CreateTestInventory(product1.ID, func(i *models.Inventory) {
			i.Available = 5
			i.MinStock = 10
			i.Product = product1
		}),
		testutil.CreateTestInventory(product2.ID, func(i *models.Inventory) {
			i.Available = 8
			i.MinStock = 15
			i.Product = product2
		}),
	}

	// Mock expectations
	suite.inventoryRepo.On("GetLowStockItems", suite.ctx, 10).Return(lowStockItems, nil)

	// Execute
	response, err := suite.inventoryService.GetLowStockAlert(suite.ctx, 10)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 10, response.Threshold)
	assert.Equal(suite.T(), 2, response.Count)
	assert.Equal(suite.T(), 2, len(response.Products))
	assert.Equal(suite.T(), "Low Stock Product 1", response.Products[0].ProductName)
	assert.Equal(suite.T(), "SKU-001", response.Products[0].SKU)
}

// Test GetLowStockAlert - No Low Stock Items
func (suite *InventoryServiceTestSuite) TestGetLowStockAlert_NoItems() {
	// Mock expectations
	suite.inventoryRepo.On("GetLowStockItems", suite.ctx, 10).Return([]*models.Inventory{}, nil)

	// Execute
	response, err := suite.inventoryService.GetLowStockAlert(suite.ctx, 10)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 0, response.Count)
	assert.Equal(suite.T(), 0, len(response.Products))
}

// Test GetLowStockAlert - Default Threshold
func (suite *InventoryServiceTestSuite) TestGetLowStockAlert_DefaultThreshold() {
	// Mock expectations (default threshold should be 10)
	suite.inventoryRepo.On("GetLowStockItems", suite.ctx, 10).Return([]*models.Inventory{}, nil)

	// Execute
	response, err := suite.inventoryService.GetLowStockAlert(suite.ctx, -1)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 10, response.Threshold)
}

// Test GetLowStockAlert - Repository Error
func (suite *InventoryServiceTestSuite) TestGetLowStockAlert_RepositoryError() {
	// Mock expectations
	suite.inventoryRepo.On("GetLowStockItems", suite.ctx, 10).Return(nil, errors.New("database error"))

	// Execute
	response, err := suite.inventoryService.GetLowStockAlert(suite.ctx, 10)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test GetLowStockAlert - Missing Product Information
func (suite *InventoryServiceTestSuite) TestGetLowStockAlert_MissingProductInfo() {
	lowStockItems := []*models.Inventory{
		testutil.CreateTestInventory("product-1", func(i *models.Inventory) {
			i.Available = 5
			i.MinStock = 10
			i.Product = nil // No product information
		}),
	}

	// Mock expectations
	suite.inventoryRepo.On("GetLowStockItems", suite.ctx, 10).Return(lowStockItems, nil)

	// Execute
	response, err := suite.inventoryService.GetLowStockAlert(suite.ctx, 10)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 1, response.Count)
	assert.Equal(suite.T(), "Unknown Product", response.Products[0].ProductName)
	assert.Equal(suite.T(), "", response.Products[0].SKU)
}

// TestInventoryServiceTestSuite runs the test suite
func TestInventoryServiceTestSuite(t *testing.T) {
	suite.Run(t, new(InventoryServiceTestSuite))
}
