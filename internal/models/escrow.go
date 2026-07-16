package models

import (
	"time"

	"github.com/google/uuid"
)

type Escrow struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID       uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"order_id"`
	BuyerID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"buyer_id"`
	SellerID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"seller_id"`
	Amount        float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	Currency      string     `gorm:"size:3;default:USD" json:"currency"`
	Status        string     `gorm:"size:50;default:held" json:"status"`
	ReleaseDate   *time.Time `json:"release_date,omitempty"`
	ReleasedAt    *time.Time `json:"released_at,omitempty"`
	PaystackRef   string     `gorm:"size:255" json:"paystack_ref,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	Order Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}
