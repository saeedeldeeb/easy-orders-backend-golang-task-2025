package services

import (
	"context"
	"errors"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/logger"
)

// productService implements ProductService interface
type productService struct {
	productRepo   repository.ProductRepository
	inventoryRepo repository.InventoryRepository
	logger        *logger.Logger
}

// NewProductService creates a new product service
func NewProductService(
	productRepo repository.ProductRepository,
	inventoryRepo repository.InventoryRepository,
	logger *logger.Logger,
) ProductService {
	return &productService{
		productRepo:   productRepo,
		inventoryRepo: inventoryRepo,
		logger:        logger,
	}
}

func (s *productService) CreateProduct(ctx context.Context, req CreateProductRequest) (*ProductResponse, error) {
	s.logger.Info("Creating product", "name", req.Name, "sku", req.SKU)

	// Validate request
	if req.Name == "" {
		return nil, errors.New("product name is required")
	}
	if req.SKU == "" {
		return nil, errors.New("product SKU is required")
	}
	if req.Price <= 0 {
		return nil, errors.New("product price must be greater than 0")
	}

	// Check if SKU already exists
	existingProduct, err := s.productRepo.GetBySKU(ctx, req.SKU)
	if err != nil {
		s.logger.Error("Failed to check existing product", "error", err, "sku", req.SKU)
		return nil, err
	}
	if existingProduct != nil {
		return nil, errors.New("product with this SKU already exists")
	}

	product := &models.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		SKU:         req.SKU,
		IsActive:    true,
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		s.logger.Error("Failed to create product", "error", err, "sku", req.SKU)
		return nil, err
	}

	// Create initial inventory if quantity is provided
	if req.InitialStock > 0 {
		// We need to create inventory via repository (this would need a repository method)
		// For now, just log it
		s.logger.Info("Initial stock would be created", "product_id", product.ID, "quantity", req.InitialStock)
	}

	s.logger.Info("Product created successfully", "id", product.ID, "sku", product.SKU)

	return &ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		SKU:         product.SKU,
		IsActive:    product.IsActive,
		Stock:       req.InitialStock, // Will be updated when we get actual inventory
	}, nil
}

func (s *productService) GetProduct(ctx context.Context, id string) (*ProductResponse, error) {
	s.logger.Debug("Getting product", "id", id)

	if id == "" {
		return nil, errors.New("product ID is required")
	}

	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get product", "error", err, "id", id)
		return nil, err
	}

	if product == nil {
		return nil, errors.New("product not found")
	}

	// Get inventory information
	inventory, err := s.inventoryRepo.GetByProductID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get product inventory", "error", err, "product_id", id)
		// Don't fail the request, just log the error
	}

	stock := 0
	if inventory != nil {
		stock = inventory.Available
	}

	return &ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		SKU:         product.SKU,
		IsActive:    product.IsActive,
		Stock:       stock,
	}, nil
}

func (s *productService) UpdateProduct(ctx context.Context, id string, req UpdateProductRequest) (*ProductResponse, error) {
	s.logger.Info("Updating product", "id", id)

	if id == "" {
		return nil, errors.New("product ID is required")
	}

	// Get existing product
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get product for update", "error", err, "id", id)
		return nil, err
	}

	if product == nil {
		return nil, errors.New("product not found")
	}

	// Update fields if provided
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.Price > 0 {
		product.Price = req.Price
	}
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	if err := s.productRepo.Update(ctx, product); err != nil {
		s.logger.Error("Failed to update product", "error", err, "id", id)
		return nil, err
	}

	s.logger.Info("Product updated successfully", "id", id)

	// Get updated inventory
	inventory, err := s.inventoryRepo.GetByProductID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get updated product inventory", "error", err, "product_id", id)
	}

	stock := 0
	if inventory != nil {
		stock = inventory.Available
	}

	return &ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		SKU:         product.SKU,
		IsActive:    product.IsActive,
		Stock:       stock,
	}, nil
}

func (s *productService) ListProducts(ctx context.Context, req ListProductsRequest) (*ListProductsResponse, error) {
	s.logger.Debug("Listing products", "page", req.Page, "limit", req.Limit)

	// Set default limit if isn't provided
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20 // Default limit
	}

	// Set the default page to 1 if not provided or invalid
	page := req.Page
	if page < 1 {
		page = 1
	}

	// Calculate offset from page number
	offset := (page - 1) * limit

	// Get paginated products
	var products []*models.Product
	var err error

	if req.ActiveOnly {
		products, err = s.productRepo.GetActive(ctx, offset, limit)
	} else {
		products, err = s.productRepo.List(ctx, offset, limit)
	}

	if err != nil {
		s.logger.Error("Failed to list products", "error", err)
		return nil, err
	}

	// Get total count
	var totalCount int64
	if req.ActiveOnly {
		totalCount, err = s.productRepo.CountActive(ctx)
	} else {
		totalCount, err = s.productRepo.Count(ctx)
	}

	if err != nil {
		s.logger.Error("Failed to count products", "error", err)
		return nil, err
	}

	// Convert to response format
	productResponses := make([]*ProductResponse, len(products))
	for i, product := range products {
		stock := 0
		if product.Inventory != nil {
			stock = product.Inventory.Available
		}

		productResponses[i] = &ProductResponse{
			ID:          product.ID,
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price,
			SKU:         product.SKU,
			IsActive:    product.IsActive,
			Stock:       stock,
		}
	}

	s.logger.Debug("Products listed successfully", "count", len(productResponses))

	return &ListProductsResponse{
		Products: productResponses,
		Page:     page,
		Limit:    limit,
		Total:    int(totalCount),
	}, nil
}
