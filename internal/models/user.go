package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username     string         `gorm:"uniqueIndex;not null;size:255" json:"username"`
	Email        string         `gorm:"uniqueIndex;not null;size:255" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
	PhoneNumber  string         `gorm:"size:50" json:"phone_number,omitempty"`
	ProfilePhoto string         `gorm:"size:500" json:"profile_photo,omitempty"`
	IsSeller     bool           `gorm:"default:false" json:"is_seller"`
	IsBuyer      bool           `gorm:"default:true" json:"is_buyer"`
	IsVerified   bool           `gorm:"default:false" json:"is_verified"`
	SellerPaid   bool           `gorm:"default:false" json:"seller_paid"`
	BusinessName string         `gorm:"size:500" json:"business_name,omitempty"`
	BusinessType string         `gorm:"size:100" json:"business_type,omitempty"`
	BusinessAddress string     `gorm:"type:text" json:"business_address,omitempty"`
	TaxID        string         `gorm:"size:100" json:"tax_id,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Organizations []Organization     `gorm:"many2many:organization_members;" json:"organizations,omitempty"`
	Products      []Product          `gorm:"foreignKey:SellerID" json:"products,omitempty"`
	Orders        []Order            `gorm:"foreignKey:BuyerID" json:"orders,omitempty"`
	Notifications []Notification     `gorm:"foreignKey:UserID" json:"notifications,omitempty"`
	CartItems     []CartItem         `gorm:"foreignKey:UserID" json:"cart_items,omitempty"`
	RFQs          []RFQ              `gorm:"foreignKey:BuyerID" json:"rfqs,omitempty"`
	Reviews       []Review           `gorm:"foreignKey:BuyerID" json:"reviews,omitempty"`
	Addresses     []Address          `gorm:"foreignKey:UserID" json:"addresses,omitempty"`
	Payments      []Payment          `gorm:"foreignKey:UserID" json:"payments,omitempty"`
}
