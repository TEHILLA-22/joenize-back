package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Shipment struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"order_id"`
	SellerID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"seller_id"`
	BuyerID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"buyer_id"`
	TrackingNumber string        `gorm:"size:255" json:"tracking_number,omitempty"`
	Carrier        string        `gorm:"size:255" json:"carrier,omitempty"`
	Status         string        `gorm:"size:50;default:pending" json:"status"`
	Origin         string        `gorm:"type:text" json:"origin,omitempty"`
	Destination    string        `gorm:"type:text" json:"destination,omitempty"`
	EstimatedDays  int           `json:"estimated_days,omitempty"`
	ShippedAt      *time.Time    `json:"shipped_at,omitempty"`
	DeliveredAt    *time.Time    `json:"delivered_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	Order  Order           `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Events []ShipmentEvent `json:"events,omitempty"`
}

type ShipmentEvent struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ShipmentID uuid.UUID `gorm:"type:uuid;not null;index" json:"shipment_id"`
	Status     string    `gorm:"size:50" json:"status"`
	Location   string    `gorm:"size:255" json:"location,omitempty"`
	Description string   `gorm:"type:text" json:"description,omitempty"`
	Timestamp  time.Time `gorm:"autoCreateTime" json:"timestamp"`
}
