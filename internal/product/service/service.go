package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"ecommerce/internal/product/domain"
	"ecommerce/internal/product/repository"
	"ecommerce/pkg/errors"
	"ecommerce/pkg/validator"
)

// ProductService defines the product service interface
type ProductService interface {
	CreateProduct(ctx context.Context, req *domain.CreateProductRequest) (*domain.Product, error)
	GetProduct(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	UpdateProduct(ctx context.Context, id uuid.UUID, req *domain.UpdateProductRequest) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id uuid.UUID) error
	ListProducts(ctx context.Context, filters *domain.ProductFilters) (*domain.ProductList, error)
	SearchProducts(ctx context.Context, query string, filters *domain.ProductFilters) (*domain.ProductList, error)
	
	CreateCategory(ctx context.Context, req *domain.CreateCategoryRequest) (*domain.Category, error)
	GetCategory(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, req *domain.UpdateCategoryRequest) (*domain.Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
	ListCategories(ctx context.Context) ([]domain.Category, error)
}

type productService struct {
	repo      repository.ProductRepository
	logger    *logrus.Logger
	validator *validator.Validator
}

// NewProductService creates a new product service
func NewProductService(repo repository.ProductRepository, logger *logrus.Logger) ProductService {
	return &productService{
		repo:      repo,
		logger:    logger,
		validator: validator.New(),
	}
}

func (s *productService) CreateProduct(ctx context.Context, req *domain.CreateProductRequest) (*domain.Product, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		s.logger.WithError(err).Error("Invalid create product request")
		return nil, errors.NewValidationError("Invalid request", err)
	}
	
	// Check if SKU already exists
	existing, err := s.repo.GetBySKU(ctx, req.SKU)
	if err != nil && !errors.IsNotFound(err) {
		s.logger.WithError(err).Error("Failed to check SKU uniqueness")
		return nil, errors.NewInternalError("Failed to validate SKU", err)
	}
	if existing != nil {
		return nil, errors.NewConflictError("SKU already exists", nil)
	}
	
	// Verify category exists
	if _, err := s.repo.GetCategory(ctx, req.CategoryID); err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFoundError("Category not found", err)
		}
		return nil, errors.NewInternalError("Failed to verify category", err)
	}
	
	product := &domain.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		CategoryID:  req.CategoryID,
		Stock:       req.Stock,
		ImageURL:    req.ImageURL,
		SKU:         req.SKU,
		IsActive:    true,
	}
	
	if err := s.repo.Create(ctx, product); err != nil {
		s.logger.WithError(err).Error("Failed to create product")
		return nil, errors.NewInternalError("Failed to create product", err)
	}
	
	// Invalidate cache
	s.repo.InvalidateProductCache(ctx)
	
	s.logger.WithField("product_id", product.ID).Info("Product created successfully")
	return product, nil
}

func (s *productService) GetProduct(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFoundError("Product not found", err)
		}
		s.logger.WithError(err).Error("Failed to get product")
		return nil, errors.NewInternalError("Failed to get product", err)
	}
	
	return product, nil
}

func (s *productService) UpdateProduct(ctx context.Context, id uuid.UUID, req *domain.UpdateProductRequest) (*domain.Product, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		s.logger.WithError(err).Error("Invalid update product request")
		return nil, errors.NewValidationError("Invalid request", err)
	}
	
	// Get existing product
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFoundError("Product not found", err)
		}
		return nil, errors.NewInternalError("Failed to get product", err)
	}
	
	// Check SKU uniqueness if being updated
	if req.SKU != nil && *req.SKU != product.SKU {
		existing, err := s.repo.GetBySKU(ctx, *req.SKU)
		if err != nil && !errors.IsNotFound(err) {
			return nil, errors.NewInternalError("Failed to validate SKU", err)
		}
		if existing != nil {
			return nil, errors.NewConflictError("SKU already exists", nil)
		}
	}
	
	// Verify category exists if being updated
	if req.CategoryID != nil {
		if _, err := s.repo.GetCategory(ctx, *req.CategoryID); err != nil {
			if errors.IsNotFound(err) {
				return nil, errors.NewNotFoundError("Category not found", err)
			}
			return nil, errors.NewInternalError("Failed to verify category", err)
		}
	}
	
	// Update fields
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.CategoryID != nil {
		product.CategoryID = *req.CategoryID
	}
	if req.Stock != nil {
		product.Stock = *req.Stock
	}
	if req.ImageURL != nil {
		product.ImageURL = *req.ImageURL
	}
	if req.SKU != nil {
		product.SKU = *req.SKU
	}
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}
	
	if err := s.repo.Update(ctx, product); err != nil {
		s.logger.WithError(err).Error("Failed to update product")
		return nil, errors.NewInternalError("Failed to update product", err)
	}
	
	// Invalidate cache
	s.repo.InvalidateProductCache(ctx)
	
	s.logger.WithField("product_id", product.ID).Info("Product updated successfully")
	return product, nil
}

func (s *productService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	// Check if product exists
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		if errors.IsNotFound(err) {
			return errors.NewNotFoundError("Product not found", err)
		}
		return errors.NewInternalError("Failed to get product", err)
	}
	
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.WithError(err).Error("Failed to delete product")
		return errors.NewInternalError("Failed to delete product", err)
	}
	
	// Invalidate cache
	s.repo.InvalidateProductCache(ctx)
	
	s.logger.WithField("product_id", id).Info("Product deleted successfully")
	return nil
}

func (s *productService) ListProducts(ctx context.Context, filters *domain.ProductFilters) (*domain.ProductList, error) {
	// Set default values
	if filters.Limit <= 0 {
		filters.Limit = 20
	}
	if filters.Limit > 100 {
		filters.Limit = 100
	}
	if filters.SortBy == "" {
		filters.SortBy = "created_at"
	}
	if filters.SortOrder == "" {
		filters.SortOrder = "desc"
	}
	
	products, total, err := s.repo.List(ctx, filters)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list products")
		return nil, errors.NewInternalError("Failed to list products", err)
	}
	
	return &domain.ProductList{
		Products: products,
		Total:    total,
		Limit:    filters.Limit,
		Offset:   filters.Offset,
		HasMore:  int64(filters.Offset+filters.Limit) < total,
	}, nil
}

func (s *productService) SearchProducts(ctx context.Context, query string, filters *domain.ProductFilters) (*domain.ProductList, error) {
	if query == "" {
		return s.ListProducts(ctx, filters)
	}
	
	// Set search query in filters
	filters.Search = query
	
	return s.ListProducts(ctx, filters)
}

func (s *productService) CreateCategory(ctx context.Context, req *domain.CreateCategoryRequest) (*domain.Category, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		s.logger.WithError(err).Error("Invalid create category request")
		return nil, errors.NewValidationError("Invalid request", err)
	}
	
	// Check if name already exists
	existing, err := s.repo.GetCategoryByName(ctx, req.Name)
	if err != nil && !errors.IsNotFound(err) {
		return nil, errors.NewInternalError("Failed to validate category name", err)
	}
	if existing != nil {
		return nil, errors.NewConflictError("Category name already exists", nil)
	}
	
	// Verify parent category exists if specified
	if req.ParentID != nil {
		if _, err := s.repo.GetCategory(ctx, *req.ParentID); err != nil {
			if errors.IsNotFound(err) {
				return nil, errors.NewNotFoundError("Parent category not found", err)
			}
			return nil, errors.NewInternalError("Failed to verify parent category", err)
		}
	}
	
	category := &domain.Category{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
		IsActive:    true,
	}
	
	if err := s.repo.CreateCategory(ctx, category); err != nil {
		s.logger.WithError(err).Error("Failed to create category")
		return nil, errors.NewInternalError("Failed to create category", err)
	}
	
	s.logger.WithField("category_id", category.ID).Info("Category created successfully")
	return category, nil
}

func (s *productService) GetCategory(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	category, err := s.repo.GetCategory(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFoundError("Category not found", err)
		}
		s.logger.WithError(err).Error("Failed to get category")
		return nil, errors.NewInternalError("Failed to get category", err)
	}
	
	return category, nil
}

func (s *productService) UpdateCategory(ctx context.Context, id uuid.UUID, req *domain.UpdateCategoryRequest) (*domain.Category, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		s.logger.WithError(err).Error("Invalid update category request")
		return nil, errors.NewValidationError("Invalid request", err)
	}
	
	// Get existing category
	category, err := s.repo.GetCategory(ctx, id)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFoundError("Category not found", err)
		}
		return nil, errors.NewInternalError("Failed to get category", err)
	}
	
	// Check name uniqueness if being updated
	if req.Name != nil && *req.Name != category.Name {
		existing, err := s.repo.GetCategoryByName(ctx, *req.Name)
		if err != nil && !errors.IsNotFound(err) {
			return nil, errors.NewInternalError("Failed to validate category name", err)
		}
		if existing != nil {
			return nil, errors.NewConflictError("Category name already exists", nil)
		}
	}
	
	// Verify parent category exists if being updated
	if req.ParentID != nil {
		if _, err := s.repo.GetCategory(ctx, *req.ParentID); err != nil {
			if errors.IsNotFound(err) {
				return nil, errors.NewNotFoundError("Parent category not found", err)
			}
			return nil, errors.NewInternalError("Failed to verify parent category", err)
		}
	}
	
	// Update fields
	if req.Name != nil {
		category.Name = *req.Name
	}
	if req.Description != nil {
		category.Description = *req.Description
	}
	if req.ParentID != nil {
		category.ParentID = req.ParentID
	}
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}
	
	if err := s.repo.UpdateCategory(ctx, category); err != nil {
		s.logger.WithError(err).Error("Failed to update category")
		return nil, errors.NewInternalError("Failed to update category", err)
	}
	
	s.logger.WithField("category_id", category.ID).Info("Category updated successfully")
	return category, nil
}

func (s *productService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	// Check if category exists
	if _, err := s.repo.GetCategory(ctx, id); err != nil {
		if errors.IsNotFound(err) {
			return errors.NewNotFoundError("Category not found", err)
		}
		return errors.NewInternalError("Failed to get category", err)
	}
	
	// Check if category has products
	filters := &domain.ProductFilters{CategoryID: &id, Limit: 1}
	products, _, err := s.repo.List(ctx, filters)
	if err != nil {
		return errors.NewInternalError("Failed to check category usage", err)
	}
	if len(products) > 0 {
		return errors.NewConflictError("Cannot delete category with products", nil)
	}
	
	if err := s.repo.DeleteCategory(ctx, id); err != nil {
		s.logger.WithError(err).Error("Failed to delete category")
		return errors.NewInternalError("Failed to delete category", err)
	}
	
	s.logger.WithField("category_id", id).Info("Category deleted successfully")
	return nil
}

func (s *productService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	categories, err := s.repo.ListCategories(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list categories")
		return nil, errors.NewInternalError("Failed to list categories", err)
	}
	
	return categories, nil
}