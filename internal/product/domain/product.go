package domain

import (
	"time"

	"github.com/google/uuid"
)

// Product represents a product in the system
type Product struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"not null" validate:"required,min=1,max=255"`
	Description string    `json:"description" gorm:"type:text"`
	Price       float64   `json:"price" gorm:"not null" validate:"required,gt=0"`
	CategoryID  uuid.UUID `json:"category_id" gorm:"type:uuid"`
	Category    *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Stock       int       `json:"stock" gorm:"default:0" validate:"gte=0"`
	ImageURL    string    `json:"image_url"`
	SKU         string    `json:"sku" gorm:"unique"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Category represents a product category
type Category struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string     `json:"name" gorm:"not null;unique" validate:"required,min=1,max=100"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id" gorm:"type:uuid"`
	Parent      *Category  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children    []Category `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateProductRequest represents the request to create a product
type CreateProductRequest struct {
	Name        string    `json:"name" validate:"required,min=1,max=255"`
	Description string    `json:"description"`
	Price       float64   `json:"price" validate:"required,gt=0"`
	CategoryID  uuid.UUID `json:"category_id" validate:"required"`
	Stock       int       `json:"stock" validate:"gte=0"`
	ImageURL    string    `json:"image_url"`
	SKU         string    `json:"sku" validate:"required"`
}

// UpdateProductRequest represents the request to update a product
type UpdateProductRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string    `json:"description,omitempty"`
	Price       *float64   `json:"price,omitempty" validate:"omitempty,gt=0"`
	CategoryID  *uuid.UUID `json:"category_id,omitempty"`
	Stock       *int       `json:"stock,omitempty" validate:"omitempty,gte=0"`
	ImageURL    *string    `json:"image_url,omitempty"`
	SKU         *string    `json:"sku,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

// ProductFilters represents filters for product queries
type ProductFilters struct {
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	MinPrice   *float64   `json:"min_price,omitempty"`
	MaxPrice   *float64   `json:"max_price,omitempty"`
	Search     string     `json:"search,omitempty"`
	IsActive   *bool      `json:"is_active,omitempty"`
	InStock    *bool      `json:"in_stock,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
	SortBy     string     `json:"sort_by,omitempty"` // name, price, created_at
	SortOrder  string     `json:"sort_order,omitempty"` // asc, desc
}

// ProductList represents a paginated list of products
type ProductList struct {
	Products   []Product `json:"products"`
	Total      int64     `json:"total"`
	Limit      int       `json:"limit"`
	Offset     int       `json:"offset"`
	HasMore    bool      `json:"has_more"`
}

// CreateCategoryRequest represents the request to create a category
type CreateCategoryRequest struct {
	Name        string     `json:"name" validate:"required,min=1,max=100"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
}

// UpdateCategoryRequest represents the request to update a category
type UpdateCategoryRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string    `json:"description,omitempty"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

// TableName returns the table name for Product
func (Product) TableName() string {
	return "products"
}

// TableName returns the table name for Category
func (Category) TableName() string {
	return "categories"
}