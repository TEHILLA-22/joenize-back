package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Category struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	Name        string         `gorm:"uniqueIndex:idx_cat_name_parent;size:255;not null" json:"name"`
	Slug        string         `gorm:"uniqueIndex:idx_cat_slug_parent;size:255;not null" json:"slug"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Parent       *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Subcategories []Category `gorm:"foreignKey:ParentID" json:"subcategories,omitempty"`
	Products     []Product  `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}

type Product struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SellerID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"seller_id"`
	OrgID       *uuid.UUID     `gorm:"type:uuid" json:"org_id,omitempty"`
	CategoryID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"category_id"`
	Name        string         `gorm:"size:500;not null" json:"name"`
	Slug        string         `gorm:"size:500;index" json:"slug"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	Price       float64        `gorm:"type:decimal(15,2)" json:"price,omitempty"`
	Currency    string         `gorm:"size:3;default:USD" json:"currency"`
	MOQ         int            `json:"moq,omitempty"`
	Stock       int            `gorm:"default:0" json:"stock"`
	InStock     bool           `gorm:"default:false" json:"in_stock"`
	Status      string         `gorm:"size:50;default:draft" json:"status"`
	IsFeatured  bool           `gorm:"default:false" json:"is_featured"`
	Tags        string         `gorm:"size:1000" json:"tags,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Seller    User          `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
	Category  Category      `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Images    []ProductImage `json:"images,omitempty"`
}

type ProductImage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProductID uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	URL       string    `gorm:"size:1000;not null" json:"url"`
	IsPrimary bool      `gorm:"default:false" json:"is_primary"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
}
