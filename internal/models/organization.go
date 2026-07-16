package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Organization struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name               string         `gorm:"size:255;not null" json:"name"`
	Description        string         `gorm:"type:text" json:"description,omitempty"`
	Logo               string         `gorm:"size:500" json:"logo,omitempty"`
	Email              string         `gorm:"size:255" json:"email,omitempty"`
	PhoneNumber        string         `gorm:"size:50" json:"phone_number,omitempty"`
	Website            string         `gorm:"size:500" json:"website,omitempty"`
	Industry           string         `gorm:"size:255" json:"industry,omitempty"`
	BusinessType       string         `gorm:"size:100" json:"business_type,omitempty"`
	RegistrationNumber string         `gorm:"size:255" json:"registration_number,omitempty"`
	TaxIDNumber        string         `gorm:"size:255" json:"tax_id_number,omitempty"`
	YearEstablished    *int           `json:"year_established,omitempty"`
	NumberOfEmployees  string         `gorm:"size:50" json:"number_of_employees,omitempty"`
	Country            string         `gorm:"size:100" json:"country,omitempty"`
	State              string         `gorm:"size:100" json:"state,omitempty"`
	City               string         `gorm:"size:100" json:"city,omitempty"`
	Address            string         `gorm:"type:text" json:"address,omitempty"`
	PostalCode         string         `gorm:"size:20" json:"postal_code,omitempty"`
	IsVerified         bool           `gorm:"default:false" json:"is_verified"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	Members  []User    `gorm:"many2many:organization_members;" json:"members,omitempty"`
	Products []Product `gorm:"foreignKey:OrgID" json:"products,omitempty"`
}

type OrganizationMember struct {
	UserID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	OrganizationID uuid.UUID `gorm:"type:uuid;primaryKey" json:"organization_id"`
	Role           string    `gorm:"size:50;default:member" json:"role"`
	JoinedAt       time.Time `gorm:"autoCreateTime" json:"joined_at"`
}
