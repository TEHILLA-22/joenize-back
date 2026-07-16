package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Order struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	BuyerID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"buyer_id"`
	SellerID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"seller_id"`
	OrgID           *uuid.UUID     `gorm:"type:uuid" json:"org_id,omitempty"`
	OrderNumber     string         `gorm:"uniqueIndex;size:50;not null" json:"order_number"`
	Status          string         `gorm:"size:50;default:pending" json:"status"`
	Subtotal        float64        `gorm:"type:decimal(15,2)" json:"subtotal"`
	ShippingCost    float64        `gorm:"type:decimal(15,2);default:0" json:"shipping_cost"`
	TaxAmount       float64        `gorm:"type:decimal(15,2);default:0" json:"tax_amount"`
	TotalAmount     float64        `gorm:"type:decimal(15,2)" json:"total_amount"`
	Currency        string         `gorm:"size:3;default:USD" json:"currency"`
	Notes           string         `gorm:"type:text" json:"notes,omitempty"`
	ShippingAddress string         `gorm:"type:text" json:"shipping_address,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	Buyer    User        `gorm:"foreignKey:BuyerID" json:"buyer,omitempty"`
	Seller   User        `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
	Items    []OrderItem `json:"items,omitempty"`
	Invoice  *Invoice    `json:"invoice,omitempty"`
	Shipment *Shipment   `json:"shipment,omitempty"`
	Escrow   *Escrow     `json:"escrow,omitempty"`
	Payments []Payment   `json:"payments,omitempty"`
}

type OrderItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index" json:"order_id"`
	ProductID uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Name      string    `gorm:"size:500;not null" json:"name"`
	Quantity  int       `gorm:"not null" json:"quantity"`
	UnitPrice float64   `gorm:"type:decimal(15,2)" json:"unit_price"`
	TotalPrice float64  `gorm:"type:decimal(15,2)" json:"total_price"`
}
