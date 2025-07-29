package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"ecommerce/internal/product/domain"
	customErrors "ecommerce/pkg/errors"
)

// ProductRepository defines the product repository interface
type ProductRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetBySKU(ctx context.Context, sku string) (*domain.Product, error)
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filters *domain.ProductFilters) ([]domain.Product, int64, error)

	CreateCategory(ctx context.Context, category *domain.Category) error
	GetCategory(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	GetCategoryByName(ctx context.Context, name string) (*domain.Category, error)
	UpdateCategory(ctx context.Context, category *domain.Category) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error
	ListCategories(ctx context.Context) ([]domain.Category, error)

	InvalidateProductCache(ctx context.Context) error
}

type productRepository struct {
	db     *gorm.DB
	redis  *redis.Client
	logger *logrus.Logger
}

// NewProductRepository creates a new product repository
func NewProductRepository(db *gorm.DB, redisClient *redis.Client, logger *logrus.Logger) ProductRepository {
	return &productRepository{
		db:     db,
		redis:  redisClient,
		logger: logger,
	}
}

func (r *productRepository) Create(ctx context.Context, product *domain.Product) error {
	if err := r.db.WithContext(ctx).Create(product).Error; err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}
	return nil
}

func (r *productRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("product:%s", id.String())
	cached, err := r.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var product domain.Product
		if err := json.Unmarshal([]byte(cached), &product); err == nil {
			return &product, nil
		}
	}

	var product domain.Product
	err = r.db.WithContext(ctx).
		Preload("Category").
		First(&product, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, customErrors.NewNotFoundError("Product not found", err)
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	// Cache the result
	if productJSON, err := json.Marshal(product); err == nil {
		r.redis.Set(ctx, cacheKey, productJSON, 10*time.Minute)
	}

	return &product, nil
}

func (r *productRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	var product domain.Product
	err := r.db.WithContext(ctx).
		Preload("Category").
		First(&product, "sku = ?", sku).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, customErrors.NewNotFoundError("Product not found", err)
		}
		return nil, fmt.Errorf("failed to get product by SKU: %w", err)
	}

	return &product, nil
}

func (r *productRepository) Update(ctx context.Context, product *domain.Product) error {
	if err := r.db.WithContext(ctx).Save(product).Error; err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("product:%s", product.ID.String())
	r.redis.Del(ctx, cacheKey)

	return nil
}

func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Product{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("product:%s", id.String())
	r.redis.Del(ctx, cacheKey)

	return nil
}

func (r *productRepository) List(ctx context.Context, filters *domain.ProductFilters) ([]domain.Product, int64, error) {
	// Try cache for common queries
	cacheKey := r.buildCacheKey(filters)
	if cacheKey != "" {
		cached, err := r.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var result struct {
				Products []domain.Product `json:"products"`
				Total    int64            `json:"total"`
			}
			if err := json.Unmarshal([]byte(cached), &result); err == nil {
				return result.Products, result.Total, nil
			}
		}
	}

	query := r.db.WithContext(ctx).Model(&domain.Product{}).Preload("Category")

	// Apply filters
	if filters.CategoryID != nil {
		query = query.Where("category_id = ?", *filters.CategoryID)
	}
	if filters.MinPrice != nil {
		query = query.Where("price >= ?", *filters.MinPrice)
	}
	if filters.MaxPrice != nil {
		query = query.Where("price <= ?", *filters.MaxPrice)
	}
	if filters.Search != "" {
		searchTerm := "%" + strings.ToLower(filters.Search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", searchTerm, searchTerm)
	}
	if filters.IsActive != nil {
		query = query.Where("is_active = ?", *filters.IsActive)
	}
	if filters.InStock != nil && *filters.InStock {
		query = query.Where("stock > 0")
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	// Apply sorting
	orderClause := fmt.Sprintf("%s %s", filters.SortBy, strings.ToUpper(filters.SortOrder))
	query = query.Order(orderClause)

	// Apply pagination
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	var products []domain.Product
	if err := query.Find(&products).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}

	// Cache the result for common queries
	if cacheKey != "" {
		result := struct {
			Products []domain.Product `json:"products"`
			Total    int64            `json:"total"`
		}{
			Products: products,
			Total:    total,
		}
		if resultJSON, err := json.Marshal(result); err == nil {
			r.redis.Set(ctx, cacheKey, resultJSON, 5*time.Minute)
		}
	}

	return products, total, nil
}

func (r *productRepository) CreateCategory(ctx context.Context, category *domain.Category) error {
	if err := r.db.WithContext(ctx).Create(category).Error; err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}
	return nil
}

func (r *productRepository) GetCategory(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	var category domain.Category
	err := r.db.WithContext(ctx).
		Preload("Parent").
		Preload("Children").
		First(&category, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, customErrors.NewNotFoundError("Category not found", err)
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	return &category, nil
}

func (r *productRepository) GetCategoryByName(ctx context.Context, name string) (*domain.Category, error) {
	var category domain.Category
	err := r.db.WithContext(ctx).First(&category, "name = ?", name).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, customErrors.NewNotFoundError("Category not found", err)
		}
		return nil, fmt.Errorf("failed to get category by name: %w", err)
	}

	return &category, nil
}

func (r *productRepository) UpdateCategory(ctx context.Context, category *domain.Category) error {
	if err := r.db.WithContext(ctx).Save(category).Error; err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}
	return nil
}

func (r *productRepository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Category{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	return nil
}

func (r *productRepository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	var categories []domain.Category
	err := r.db.WithContext(ctx).
		Preload("Parent").
		Preload("Children").
		Where("is_active = ?", true).
		Order("name ASC").
		Find(&categories).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}

	return categories, nil
}

func (r *productRepository) InvalidateProductCache(ctx context.Context) error {
	// Delete all product-related cache keys
	keys, err := r.redis.Keys(ctx, "product:*").Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return r.redis.Del(ctx, keys...).Err()
	}

	// Also delete list cache keys
	listKeys, err := r.redis.Keys(ctx, "products:*").Result()
	if err != nil {
		return err
	}

	if len(listKeys) > 0 {
		return r.redis.Del(ctx, listKeys...).Err()
	}

	return nil
}

func (r *productRepository) buildCacheKey(filters *domain.ProductFilters) string {
	// Only cache simple queries to avoid cache explosion
	if filters.Search != "" || filters.MinPrice != nil || filters.MaxPrice != nil {
		return ""
	}

	key := "products:list"
	if filters.CategoryID != nil {
		key += fmt.Sprintf(":cat_%s", filters.CategoryID.String())
	}
	if filters.IsActive != nil {
		key += fmt.Sprintf(":active_%t", *filters.IsActive)
	}
	if filters.InStock != nil {
		key += fmt.Sprintf(":stock_%t", *filters.InStock)
	}
	key += fmt.Sprintf(":limit_%d:offset_%d", filters.Limit, filters.Offset)
	key += fmt.Sprintf(":sort_%s_%s", filters.SortBy, filters.SortOrder)

	return key
}
