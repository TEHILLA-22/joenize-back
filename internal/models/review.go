package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Review struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID   uuid.UUID      `gorm:"type:uuid;index" json:"order_id"`
	SellerID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"seller_id"`
	BuyerID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"buyer_id"`
	Rating    int            `gorm:"not null;default:5" json:"rating"`
	Comment   string         `gorm:"type:text" json:"comment,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Order  Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Seller User  `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
	Buyer  User  `gorm:"foreignKey:BuyerID" json:"buyer,omitempty"`
}

type Address struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Label      string         `gorm:"size:100" json:"label,omitempty"`
	Street     string         `gorm:"type:text;not null" json:"street"`
	City       string         `gorm:"size:100;not null" json:"city"`
	State      string         `gorm:"size:100" json:"state,omitempty"`
	Country    string         `gorm:"size:100;not null" json:"country"`
	PostalCode string         `gorm:"size:20" json:"postal_code,omitempty"`
	IsDefault  bool           `gorm:"default:false" json:"is_default"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
