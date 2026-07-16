package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Payment struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	OrderID       *uuid.UUID     `gorm:"type:uuid" json:"order_id,omitempty"`
	Reference     string         `gorm:"uniqueIndex;size:255;not null" json:"reference"`
	Amount        float64        `gorm:"type:decimal(15,2);not null" json:"amount"`
	Currency      string         `gorm:"size:3;default:USD" json:"currency"`
	Status        string         `gorm:"size:50;default:pending" json:"status"`
	Channel       string         `gorm:"size:100" json:"channel,omitempty"`
	PaidAt        *time.Time     `json:"paid_at,omitempty"`
	Metadata      string         `gorm:"type:text" json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	User   User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type Wallet struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID  uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	Balance float64   `gorm:"type:decimal(15,2);default:0" json:"balance"`
	Currency string   `gorm:"size:3;default:USD" json:"currency"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type Invoice struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID      uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"order_id"`
	InvoiceNumber string        `gorm:"uniqueIndex;size:50;not null" json:"invoice_number"`
	Amount       float64        `gorm:"type:decimal(15,2);not null" json:"amount"`
	Status       string         `gorm:"size:50;default:pending" json:"status"`
	DueDate      *time.Time     `json:"due_date,omitempty"`
	PaidAt       *time.Time     `json:"paid_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Order Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}
