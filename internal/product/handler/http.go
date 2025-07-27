package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"ecommerce/internal/product/domain"
	"ecommerce/internal/product/service"
	"ecommerce/pkg/errors"
	"ecommerce/pkg/response"
)

// HTTPHandler handles HTTP requests for product service
type HTTPHandler struct {
	service service.ProductService
	logger  *logrus.Logger
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(service service.ProductService, logger *logrus.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *HTTPHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	
	// Product routes
	products := api.Group("/products")
	{
		products.POST("", h.CreateProduct)
		products.GET("", h.ListProducts)
		products.GET("/search", h.SearchProducts)
		products.GET("/:id", h.GetProduct)
		products.PUT("/:id", h.UpdateProduct)
		products.DELETE("/:id", h.DeleteProduct)
	}
	
	// Category routes
	categories := api.Group("/categories")
	{
		categories.POST("", h.CreateCategory)
		categories.GET("", h.ListCategories)
		categories.GET("/:id", h.GetCategory)
		categories.PUT("/:id", h.UpdateCategory)
		categories.DELETE("/:id", h.DeleteCategory)
	}
	
	// Health check
	router.GET("/health", h.HealthCheck)
	router.GET("/ready", h.ReadinessCheck)
}

// CreateProduct handles product creation
func (h *HTTPHandler) CreateProduct(c *gin.Context) {
	var req domain.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	product, err := h.service.CreateProduct(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	response.Success(c, http.StatusCreated, "Product created successfully", product)
}

// GetProduct handles getting a single product
func (h *HTTPHandler) GetProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}
	
	product, err := h.service.GetProduct(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	response.Success(c, http.StatusOK, "Product retrieved successfully", product)
}

// UpdateProduct handles product updates
func (h *HTTPHandler) UpdateProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}
	
	var req domain.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	product, err := h.service.UpdateProduct(c.Request.Context(), id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	response.Success(c, http.StatusOK, "Product updated successfully", product)
}

// DeleteProduct handles product deletion
func (h *HTTPHandler) DeleteProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}
	
	if err := h.service.DeleteProduct(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	
	response.Success(c, http.StatusOK, "Product deleted successfully", nil)
}

// ListProducts handles product listing with filters
func (h *HTTPHandler) ListProducts(c *gin.Context) {
	filters := &domain.ProductFilters{}
	
	// Parse query parameters
	if categoryID := c.Query("category_id"); categoryID != "" {
		if id, err := uuid.Parse(categoryID); err == nil {
			filters.CategoryID = &id
		}
	}
	
	if minPrice := c.Query("min_price"); minPrice != "" {
		if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
			filters.MinPrice = &price
		}
	}
	
	if maxPrice := c.Query("max_price"); maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			filters.MaxPrice = &price
		}
	}
	
	filters.Search = c.Query("search")
	
	if isActive := c.Query("is_active"); isActive != "" {
		if active, err := strconv.ParseBool(isActive); err == nil {
			filters.IsActive = &active
		}
	}
	
	if inStock := c.Query("in_stock"); inStock != "" {
		if stock, err := strconv.ParseBool(inStock); err == nil {
			filters.InStock = &stock
		}
	}
	
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters.Limit = l
		}
	}
	
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters.Offset = o
		}
	}
	
	filters.SortBy = c.DefaultQuery("sort_by", "created_at")
	filters.SortOrder = c.DefaultQuery("sort_order", "desc")
	
	productList, err := h.service.ListProducts(c.Request.Context(), filters)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	response.Success(c, http.StatusOK, "Products retrieved successfully", productList)
}

// SearchProducts handles product search
func (h *HTTPHandler) SearchProducts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		response.Error(c, http.StatusBadRequest, "Search query is required", nil)
		return
	}
	
	filters := &domain.ProductFilters{}
	
	// Parse additional filters
	if categoryID := c.Query("category_id"); categoryID != "" {
		if id, err := uuid.Parse(categoryID); err == nil {
			filters.CategoryID = &id
		}
	}
	
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters.Limit = l
		}
	}
	
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters.Offset = o
		}
	}
	
	productList, err := h.service.SearchProducts(c.Request.Context(), query, filters)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	response.Success(c, http.StatusOK, "Search results retrieved successfully", productList)
}

// CreateCategory handles category creation
func (h *HTTPHandler) CreateCategory(c *gin.Context) {
	var req domain.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	category, err := h.service.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	response.Success(c, http.StatusCreated, "Category created successfully", category)
}

// GetCategory handles getting a single category
func (h *HTTPHandler) GetCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid category ID", err)
		return
	}
	
	category, err := h.service.GetCategory(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	response.Success(c, http.StatusOK, "Category retrieved successfully", category)
}

// UpdateCategory handles category updates
func (h *HTTPHandler) UpdateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid category ID", err)
		return
	}
	
	var req domain.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request body")
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	category, err := h.service.UpdateCategory(c.Request.Context(), id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	response.Success(c, http.StatusOK, "Category updated successfully", category)
}

// DeleteCategory handles category deletion
func (h *HTTPHandler) DeleteCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid category ID", err)
		return
	}
	
	if err := h.service.DeleteCategory(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	
	response.Success(c, http.StatusOK, "Category deleted successfully", nil)
}

// ListCategories handles category listing
func (h *HTTPHandler) ListCategories(c *gin.Context) {
	categories, err := h.service.ListCategories(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	
	response.Success(c, http.StatusOK, "Categories retrieved successfully", categories)
}

// HealthCheck handles health check requests
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	response.Success(c, http.StatusOK, "Service is healthy", gin.H{
		"service": "product-service",
		"status":  "healthy",
	})
}

// ReadinessCheck handles readiness check requests
func (h *HTTPHandler) ReadinessCheck(c *gin.Context) {
	// TODO: Add actual readiness checks (database, redis connectivity)
	response.Success(c, http.StatusOK, "Service is ready", gin.H{
		"service": "product-service",
		"status":  "ready",
	})
}

// handleError handles service errors and converts them to appropriate HTTP responses
func (h *HTTPHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.IsNotFound(err):
		response.Error(c, http.StatusNotFound, "Resource not found", err)
	case errors.IsValidation(err):
		response.Error(c, http.StatusBadRequest, "Validation failed", err)
	case errors.IsConflict(err):
		response.Error(c, http.StatusConflict, "Resource conflict", err)
	default:
		h.logger.WithError(err).Error("Internal server error")
		response.Error(c, http.StatusInternalServerError, "Internal server error", nil)
	}
}