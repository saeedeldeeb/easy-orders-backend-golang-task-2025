package repository

import (
	"context"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/logger"

	"gorm.io/gorm"
)

// productRepository implements ProductRepository interface
type productRepository struct {
	db     *database.DB
	logger *logger.Logger
}

// NewProductRepository creates a new product repository
func NewProductRepository(db *database.DB, logger *logger.Logger) ProductRepository {
	return &productRepository{
		db:     db,
		logger: logger,
	}
}

func (r *productRepository) Create(ctx context.Context, product *models.Product) error {
	r.logger.Debug("Creating product in database", "name", product.Name, "sku", product.SKU)

	if err := r.db.WithContext(ctx).Create(product).Error; err != nil {
		r.logger.Error("Failed to create product", "error", err, "sku", product.SKU)
		return err
	}

	r.logger.Info("Product created in database", "id", product.ID, "sku", product.SKU)
	return nil
}

func (r *productRepository) GetByID(ctx context.Context, id string) (*models.Product, error) {
	r.logger.Debug("Getting product by ID", "id", id)

	var product models.Product
	if err := r.db.WithContext(ctx).
		Preload("Inventory").
		First(&product, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Debug("Product not found", "id", id)
			return nil, nil
		}
		r.logger.Error("Failed to get product by ID", "error", err, "id", id)
		return nil, err
	}

	r.logger.Debug("Product retrieved from database", "id", id)
	return &product, nil
}

func (r *productRepository) GetBySKU(ctx context.Context, sku string) (*models.Product, error) {
	r.logger.Debug("Getting product by SKU", "sku", sku)

	var product models.Product
	if err := r.db.WithContext(ctx).
		Preload("Inventory").
		First(&product, "sku = ?", sku).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Debug("Product not found", "sku", sku)
			return nil, nil
		}
		r.logger.Error("Failed to get product by SKU", "error", err, "sku", sku)
		return nil, err
	}

	r.logger.Debug("Product retrieved from database", "sku", sku)
	return &product, nil
}

func (r *productRepository) Update(ctx context.Context, product *models.Product) error {
	r.logger.Debug("Updating product in database", "id", product.ID)

	if err := r.db.WithContext(ctx).Save(product).Error; err != nil {
		r.logger.Error("Failed to update product", "error", err, "id", product.ID)
		return err
	}

	r.logger.Info("Product updated in database", "id", product.ID)
	return nil
}

func (r *productRepository) Delete(ctx context.Context, id string) error {
	r.logger.Debug("Deleting product from database", "id", id)

	// Soft delete the product
	if err := r.db.WithContext(ctx).Delete(&models.Product{}, "id = ?", id).Error; err != nil {
		r.logger.Error("Failed to delete product", "error", err, "id", id)
		return err
	}

	r.logger.Info("Product deleted from database", "id", id)
	return nil
}

func (r *productRepository) List(ctx context.Context, offset, limit int) ([]*models.Product, error) {
	r.logger.Debug("Listing products from database", "offset", offset, "limit", limit)

	var products []*models.Product
	if err := r.db.WithContext(ctx).
		Preload("Inventory").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&products).Error; err != nil {
		r.logger.Error("Failed to list products", "error", err)
		return nil, err
	}

	r.logger.Debug("Products retrieved from database", "count", len(products))
	return products, nil
}

func (r *productRepository) Search(ctx context.Context, query string, offset, limit int) ([]*models.Product, error) {
	r.logger.Debug("Searching products", "query", query, "offset", offset, "limit", limit)

	var products []*models.Product
	searchPattern := "%" + query + "%"

	if err := r.db.WithContext(ctx).
		Preload("Inventory").
		Where("name ILIKE ? OR description ILIKE ? OR sku ILIKE ?", searchPattern, searchPattern, searchPattern).
		Where("is_active = ?", true).
		Offset(offset).
		Limit(limit).
		Order("name ASC").
		Find(&products).Error; err != nil {
		r.logger.Error("Failed to search products", "error", err, "query", query)
		return nil, err
	}

	r.logger.Debug("Products search results retrieved", "count", len(products), "query", query)
	return products, nil
}

func (r *productRepository) GetActive(ctx context.Context, offset, limit int) ([]*models.Product, error) {
	r.logger.Debug("Getting active products", "offset", offset, "limit", limit)

	var products []*models.Product
	if err := r.db.WithContext(ctx).
		Preload("Inventory").
		Where("is_active = ?", true).
		Offset(offset).
		Limit(limit).
		Order("name ASC").
		Find(&products).Error; err != nil {
		r.logger.Error("Failed to get active products", "error", err)
		return nil, err
	}

	r.logger.Debug("Active products retrieved from database", "count", len(products))
	return products, nil
}
