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

// ProductServiceTestSuite defines the test suite for ProductService
type ProductServiceTestSuite struct {
	suite.Suite
	productService services.ProductService
	productRepo    *mocks.MockProductRepository
	inventoryRepo  *mocks.MockInventoryRepository
	logger         *logger.Logger
	ctx            context.Context
}

// SetupTest runs before each test in the suite
func (suite *ProductServiceTestSuite) SetupTest() {
	suite.productRepo = new(mocks.MockProductRepository)
	suite.inventoryRepo = new(mocks.MockInventoryRepository)
	suite.logger = &logger.Logger{SugaredLogger: mocks.NewNoOpLogger()}
	suite.ctx = context.Background()

	suite.productService = services.NewProductService(
		suite.productRepo,
		suite.inventoryRepo,
		suite.logger,
	)
}

// TearDownTest runs after each test in the suite
func (suite *ProductServiceTestSuite) TearDownTest() {
	suite.productRepo.AssertExpectations(suite.T())
	suite.inventoryRepo.AssertExpectations(suite.T())
}

// Test CreateProduct - Happy Path
func (suite *ProductServiceTestSuite) TestCreateProduct_Success() {
	req := services.CreateProductRequest{
		Name:         "Test Product",
		Description:  "Test Description",
		Price:        99.99,
		SKU:          "TEST-SKU-001",
		InitialStock: 100,
		MinStock:     10,
		MaxStock:     1000,
	}

	// Mock expectations
	suite.productRepo.On("GetBySKU", suite.ctx, req.SKU).Return(nil, nil)
	suite.productRepo.On("CreateWithInventory", suite.ctx, mock.AnythingOfType("*models.Product"), mock.AnythingOfType("*models.Inventory")).
		Run(func(args mock.Arguments) {
			product := args.Get(1).(*models.Product)
			product.ID = "product-id-123"
		}).
		Return(nil)

	// Execute
	response, err := suite.productService.CreateProduct(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), "Test Product", response.Name)
	assert.Equal(suite.T(), "TEST-SKU-001", response.SKU)
	assert.Equal(suite.T(), 99.99, response.Price)
	assert.Equal(suite.T(), 100, response.Stock)
	assert.True(suite.T(), response.IsActive)
}

// Test CreateProduct - Without InitialStock
func (suite *ProductServiceTestSuite) TestCreateProduct_WithoutInitialStock() {
	req := services.CreateProductRequest{
		Name:        "Test Product",
		Description: "Test Description",
		Price:       99.99,
		SKU:         "TEST-SKU-002",
	}

	// Mock expectations
	suite.productRepo.On("GetBySKU", suite.ctx, req.SKU).Return(nil, nil)
	suite.productRepo.On("CreateWithInventory", suite.ctx, mock.AnythingOfType("*models.Product"), (*models.Inventory)(nil)).
		Run(func(args mock.Arguments) {
			product := args.Get(1).(*models.Product)
			product.ID = "product-id-456"
		}).
		Return(nil)

	// Execute
	response, err := suite.productService.CreateProduct(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 0, response.Stock)
}

// Test CreateProduct - Validation Error: Name Required
func (suite *ProductServiceTestSuite) TestCreateProduct_ValidationError_NameRequired() {
	req := services.CreateProductRequest{
		Name:  "",
		Price: 99.99,
		SKU:   "TEST-SKU",
	}

	// Execute
	response, err := suite.productService.CreateProduct(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "name is required")
}

// Test CreateProduct - Validation Error: SKU Required
func (suite *ProductServiceTestSuite) TestCreateProduct_ValidationError_SKURequired() {
	req := services.CreateProductRequest{
		Name:  "Test Product",
		Price: 99.99,
		SKU:   "",
	}

	// Execute
	response, err := suite.productService.CreateProduct(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "SKU is required")
}

// Test CreateProduct - Validation Error: Invalid Price
func (suite *ProductServiceTestSuite) TestCreateProduct_ValidationError_InvalidPrice() {
	req := services.CreateProductRequest{
		Name:  "Test Product",
		Price: 0,
		SKU:   "TEST-SKU",
	}

	// Execute
	response, err := suite.productService.CreateProduct(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "price must be greater than 0")
}

// Test CreateProduct - Duplicate SKU
func (suite *ProductServiceTestSuite) TestCreateProduct_DuplicateSKU() {
	req := services.CreateProductRequest{
		Name:  "Test Product",
		Price: 99.99,
		SKU:   "DUPLICATE-SKU",
	}

	existingProduct := testutil.CreateTestProduct()

	// Mock expectations
	suite.productRepo.On("GetBySKU", suite.ctx, req.SKU).Return(existingProduct, nil)

	// Execute
	response, err := suite.productService.CreateProduct(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "SKU already exists")
}

// Test CreateProduct - Repository Error on GetBySKU
func (suite *ProductServiceTestSuite) TestCreateProduct_RepositoryError_GetBySKU() {
	req := services.CreateProductRequest{
		Name:  "Test Product",
		Price: 99.99,
		SKU:   "TEST-SKU",
	}

	// Mock expectations
	suite.productRepo.On("GetBySKU", suite.ctx, req.SKU).Return(nil, errors.New("database error"))

	// Execute
	response, err := suite.productService.CreateProduct(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test CreateProduct - Repository Error on CreateWithInventory
func (suite *ProductServiceTestSuite) TestCreateProduct_RepositoryError_CreateWithInventory() {
	req := services.CreateProductRequest{
		Name:  "Test Product",
		Price: 99.99,
		SKU:   "TEST-SKU",
	}

	// Mock expectations
	suite.productRepo.On("GetBySKU", suite.ctx, req.SKU).Return(nil, nil)
	suite.productRepo.On("CreateWithInventory", suite.ctx, mock.AnythingOfType("*models.Product"), (*models.Inventory)(nil)).
		Return(errors.New("database error"))

	// Execute
	response, err := suite.productService.CreateProduct(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test GetProduct - Happy Path
func (suite *ProductServiceTestSuite) TestGetProduct_Success() {
	productID := "product-id-123"
	product := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = productID
	})
	inventory := testutil.CreateTestInventory(productID, func(i *models.Inventory) {
		i.Available = 50
	})

	// Mock expectations
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(product, nil)
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(inventory, nil)

	// Execute
	response, err := suite.productService.GetProduct(suite.ctx, productID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), productID, response.ID)
	assert.Equal(suite.T(), product.Name, response.Name)
	assert.Equal(suite.T(), 50, response.Stock)
}

// Test GetProduct - Without Inventory
func (suite *ProductServiceTestSuite) TestGetProduct_WithoutInventory() {
	productID := "product-id-456"
	product := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = productID
	})

	// Mock expectations
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(product, nil)
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(nil, errors.New("not found"))

	// Execute
	response, err := suite.productService.GetProduct(suite.ctx, productID)

	// Assert
	assert.NoError(suite.T(), err) // Should not fail even if inventory fetch fails
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 0, response.Stock)
}

// Test GetProduct - Validation Error: ID Required
func (suite *ProductServiceTestSuite) TestGetProduct_ValidationError_IDRequired() {
	// Execute
	response, err := suite.productService.GetProduct(suite.ctx, "")

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "ID is required")
}

// Test GetProduct - Product Not Found
func (suite *ProductServiceTestSuite) TestGetProduct_NotFound() {
	productID := "non-existent-id"

	// Mock expectations
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(nil, nil)

	// Execute
	response, err := suite.productService.GetProduct(suite.ctx, productID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// Test GetProduct - Repository Error
func (suite *ProductServiceTestSuite) TestGetProduct_RepositoryError() {
	productID := "product-id-789"

	// Mock expectations
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(nil, errors.New("database error"))

	// Execute
	response, err := suite.productService.GetProduct(suite.ctx, productID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test UpdateProduct - Happy Path
func (suite *ProductServiceTestSuite) TestUpdateProduct_Success() {
	productID := "product-id-123"
	product := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = productID
		p.Name = "Old Name"
		p.Price = 50.00
	})
	inventory := testutil.CreateTestInventory(productID)

	isActive := false
	req := services.UpdateProductRequest{
		Name:        "Updated Name",
		Description: "Updated Description",
		Price:       99.99,
		IsActive:    &isActive,
	}

	// Mock expectations
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(product, nil)
	suite.productRepo.On("Update", suite.ctx, mock.AnythingOfType("*models.Product")).Return(nil)
	suite.inventoryRepo.On("GetByProductID", suite.ctx, productID).Return(inventory, nil)

	// Execute
	response, err := suite.productService.UpdateProduct(suite.ctx, productID, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), "Updated Name", response.Name)
	assert.Equal(suite.T(), "Updated Description", response.Description)
	assert.Equal(suite.T(), 99.99, response.Price)
	assert.False(suite.T(), response.IsActive)
}

// Test UpdateProduct - Validation Error: ID Required
func (suite *ProductServiceTestSuite) TestUpdateProduct_ValidationError_IDRequired() {
	req := services.UpdateProductRequest{
		Name: "Updated Name",
	}

	// Execute
	response, err := suite.productService.UpdateProduct(suite.ctx, "", req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "ID is required")
}

// Test UpdateProduct - Product Not Found
func (suite *ProductServiceTestSuite) TestUpdateProduct_NotFound() {
	productID := "non-existent-id"
	req := services.UpdateProductRequest{
		Name: "Updated Name",
	}

	// Mock expectations
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(nil, nil)

	// Execute
	response, err := suite.productService.UpdateProduct(suite.ctx, productID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "not found")
}

// Test UpdateProduct - Repository Error on Update
func (suite *ProductServiceTestSuite) TestUpdateProduct_RepositoryError() {
	productID := "product-id-123"
	product := testutil.CreateTestProduct(func(p *models.Product) {
		p.ID = productID
	})
	req := services.UpdateProductRequest{
		Name: "Updated Name",
	}

	// Mock expectations
	suite.productRepo.On("GetByID", suite.ctx, productID).Return(product, nil)
	suite.productRepo.On("Update", suite.ctx, mock.AnythingOfType("*models.Product")).Return(errors.New("database error"))

	// Execute
	response, err := suite.productService.UpdateProduct(suite.ctx, productID, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test ListProducts - Happy Path
func (suite *ProductServiceTestSuite) TestListProducts_Success() {
	products := []*models.Product{
		testutil.CreateTestProduct(func(p *models.Product) { p.ID = "1" }),
		testutil.CreateTestProduct(func(p *models.Product) { p.ID = "2" }),
	}

	req := services.ListProductsRequest{
		Page:       1,
		Limit:      20,
		ActiveOnly: false,
	}

	// Mock expectations
	suite.productRepo.On("List", suite.ctx, 0, 20).Return(products, nil)
	suite.productRepo.On("Count", suite.ctx).Return(int64(2), nil)

	// Execute
	response, err := suite.productService.ListProducts(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 2, len(response.Products))
	assert.Equal(suite.T(), 1, response.Page)
	assert.Equal(suite.T(), 20, response.Limit)
	assert.Equal(suite.T(), 2, response.Total)
}

// Test ListProducts - ActiveOnly Filter
func (suite *ProductServiceTestSuite) TestListProducts_ActiveOnly() {
	products := []*models.Product{
		testutil.CreateTestProduct(func(p *models.Product) { p.ID = "1"; p.IsActive = true }),
	}

	req := services.ListProductsRequest{
		Page:       1,
		Limit:      20,
		ActiveOnly: true,
	}

	// Mock expectations
	suite.productRepo.On("GetActive", suite.ctx, 0, 20).Return(products, nil)
	suite.productRepo.On("CountActive", suite.ctx).Return(int64(1), nil)

	// Execute
	response, err := suite.productService.ListProducts(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 1, len(response.Products))
}

// Test ListProducts - Default Pagination
func (suite *ProductServiceTestSuite) TestListProducts_DefaultPagination() {
	products := []*models.Product{}

	req := services.ListProductsRequest{
		Page:  0, // Invalid, should default to 1
		Limit: 0, // Invalid, should default to 20
	}

	// Mock expectations (offset should be 0 for page 1, limit should be 20)
	suite.productRepo.On("List", suite.ctx, 0, 20).Return(products, nil)
	suite.productRepo.On("Count", suite.ctx).Return(int64(0), nil)

	// Execute
	response, err := suite.productService.ListProducts(suite.ctx, req)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), 1, response.Page)
	assert.Equal(suite.T(), 20, response.Limit)
}

// Test ListProducts - Repository Error on List
func (suite *ProductServiceTestSuite) TestListProducts_RepositoryError_List() {
	req := services.ListProductsRequest{
		Page:  1,
		Limit: 20,
	}

	// Mock expectations
	suite.productRepo.On("List", suite.ctx, 0, 20).Return(nil, errors.New("database error"))

	// Execute
	response, err := suite.productService.ListProducts(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// Test ListProducts - Repository Error on Count
func (suite *ProductServiceTestSuite) TestListProducts_RepositoryError_Count() {
	products := []*models.Product{}
	req := services.ListProductsRequest{
		Page:  1,
		Limit: 20,
	}

	// Mock expectations
	suite.productRepo.On("List", suite.ctx, 0, 20).Return(products, nil)
	suite.productRepo.On("Count", suite.ctx).Return(int64(0), errors.New("database error"))

	// Execute
	response, err := suite.productService.ListProducts(suite.ctx, req)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), response)
	assert.Contains(suite.T(), err.Error(), "database error")
}

// TestProductServiceTestSuite runs the test suite
func TestProductServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProductServiceTestSuite))
}
