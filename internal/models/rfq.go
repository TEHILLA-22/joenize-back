package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RFQ struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	BuyerID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"buyer_id"`
	OrgID       *uuid.UUID     `gorm:"type:uuid" json:"org_id,omitempty"`
	CategoryID  *uuid.UUID     `gorm:"type:uuid;index" json:"category_id,omitempty"`
	Title       string         `gorm:"size:500;not null" json:"title"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	TargetPrice float64        `gorm:"type:decimal(15,2)" json:"target_price,omitempty"`
	Quantity    int            `gorm:"default:1" json:"quantity"`
	Unit        string         `gorm:"size:50" json:"unit,omitempty"`
	IsPrivate   bool           `gorm:"default:false" json:"is_private"`
	Status      string         `gorm:"size:50;default:open" json:"status"`
	Notes       string         `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Buyer    User          `gorm:"foreignKey:BuyerID" json:"buyer,omitempty"`
	Category *Category     `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Items    []RFQItem     `json:"items,omitempty"`
	Images   []RFQImage    `json:"images,omitempty"`
	Quotes   []Quote       `json:"quotes,omitempty"`
	Invitations []RFQInvitation `json:"invitations,omitempty"`
}

type RFQInvitation struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RFQID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"rfq_id"`
	SupplierID uuid.UUID      `gorm:"type:uuid;not null;index" json:"supplier_id"`
	Status     string         `gorm:"size:50;default:pending" json:"status"`
	CreatedAt  time.Time      `json:"created_at"`

	RFQ      RFQ  `gorm:"foreignKey:RFQID" json:"-"`
	Supplier User `gorm:"foreignKey:SupplierID" json:"supplier,omitempty"`
}

type RFQImage struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RFQID   uuid.UUID `gorm:"type:uuid;not null;index" json:"rfq_id"`
	URL     string    `gorm:"size:500;not null" json:"url"`
	SortOrder int     `gorm:"default:0" json:"sort_order"`
}

type RFQItem struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RFQID       uuid.UUID `gorm:"type:uuid;not null;index" json:"rfq_id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	ProductName string    `gorm:"size:500" json:"product_name"`
	Quantity    int       `gorm:"not null" json:"quantity"`
	Notes       string    `gorm:"type:text" json:"notes,omitempty"`
}

type Quote struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RFQID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"rfq_id"`
	SellerID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"seller_id"`
	Amount    float64        `gorm:"type:decimal(15,2)" json:"amount"`
	Currency  string         `gorm:"size:3;default:USD" json:"currency"`
	Notes     string         `gorm:"type:text" json:"notes,omitempty"`
	Status    string         `gorm:"size:50;default:pending" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	RFQ    RFQ  `gorm:"foreignKey:RFQID" json:"rfq,omitempty"`
	Seller User `gorm:"foreignKey:SellerID" json:"seller,omitempty"`
}
