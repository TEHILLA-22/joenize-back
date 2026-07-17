package database

import (
	"fmt"
	"log"

	"github.com/tehilla-22/b2b-api/internal/config"
	"github.com/tehilla-22/b2b-api/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) *gorm.DB {
	dsn := cfg.DSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	fmt.Println("Database connected successfully")
	DB = db
	return db
}

func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.OrganizationMember{},
		&models.Category{},
		&models.Product{},
		&models.ProductImage{},
		&models.CartItem{},
		&models.RFQ{},
		&models.RFQItem{},
		&models.RFQImage{},
		&models.RFQInvitation{},
		&models.Quote{},
		&models.Order{},
		&models.OrderItem{},
		&models.Invoice{},
		&models.Payment{},
		&models.Wallet{},
		&models.Escrow{},
		&models.Shipment{},
		&models.ShipmentEvent{},
		&models.Notification{},
		&models.Review{},
		&models.Address{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}
	fmt.Println("Database migration completed")
}

func Seed(db *gorm.DB) {
	var count int64
	db.Model(&models.Category{}).Count(&count)
	if count > 0 {
		return
	}

	parents := []models.Category{
		{Name: "Electronics", Slug: "electronics", Description: "Consumer electronics, components, and devices for business procurement"},
		{Name: "Industrial Equipment", Slug: "industrial-equipment", Description: "Heavy machinery, manufacturing equipment, and industrial tools"},
		{Name: "Packaging", Slug: "packaging", Description: "Packaging materials, containers, and labeling solutions"},
		{Name: "Safety Supplies", Slug: "safety-supplies", Description: "PPE, safety gear, and workplace safety equipment"},
		{Name: "Office Operations", Slug: "office-operations", Description: "Office supplies, furniture, and operational essentials"},
		{Name: "Raw Materials", Slug: "raw-materials", Description: "Raw materials, chemicals, and base production inputs"},
		{Name: "Food & Beverage", Slug: "food-beverage", Description: "Food products, beverages, and ingredients for business"},
		{Name: "Clothing & Beauty", Slug: "clothing-beauty", Description: "Apparel, textiles, beauty products, and personal care"},
		{Name: "Automotive", Slug: "automotive", Description: "Vehicle parts, accessories, and automotive equipment"},
		{Name: "Agriculture", Slug: "agriculture", Description: "Farm supplies, equipment, and agricultural products"},
		{Name: "Construction", Slug: "construction", Description: "Building materials, construction equipment, and supplies"},
		{Name: "Healthcare", Slug: "healthcare", Description: "Medical supplies, equipment, and healthcare products"},
		{Name: "Logistics", Slug: "logistics", Description: "Shipping supplies, warehousing, and logistics solutions"},
		{Name: "Technology", Slug: "technology", Description: "Software, IT infrastructure, and technology services"},
	}
	for _, c := range parents {
		db.Create(&c)
	}

	subs := map[string][]models.Category{
		"electronics": {
			{Name: "Consumer Electronics", Slug: "consumer-electronics", Description: "Phones, tablets, laptops, and personal devices"},
			{Name: "Industrial Electronics", Slug: "industrial-electronics", Description: "Circuit boards, sensors, and industrial components"},
			{Name: "Electronic Components", Slug: "electronic-components", Description: "Chips, connectors, cables, and parts"},
		},
		"clothing-beauty": {
			{Name: "Apparel", Slug: "apparel", Description: "Clothing, uniforms, and workwear"},
			{Name: "Beauty & Cosmetics", Slug: "beauty-cosmetics", Description: "Skincare, makeup, and personal care products"},
			{Name: "Textiles & Fabrics", Slug: "textiles-fabrics", Description: "Raw fabrics, materials, and trims"},
		},
		"food-beverage": {
			{Name: "Beverages", Slug: "beverages", Description: "Soft drinks, juices, and bottled water"},
			{Name: "Ingredients", Slug: "ingredients", Description: "Raw food ingredients and additives"},
			{Name: "Processed Foods", Slug: "processed-foods", Description: "Packaged and prepared food products"},
		},
		"healthcare": {
			{Name: "Medical Equipment", Slug: "medical-equipment", Description: "Diagnostic and therapeutic equipment"},
			{Name: "Pharmaceuticals", Slug: "pharmaceuticals", Description: "Medicines and pharmaceutical products"},
			{Name: "Medical Supplies", Slug: "medical-supplies", Description: "Consumables, PPE, and disposables"},
		},
		"construction": {
			{Name: "Building Materials", Slug: "building-materials", Description: "Cement, steel, lumber, and bricks"},
			{Name: "Tools & Hardware", Slug: "tools-hardware", Description: "Hand tools, power tools, and hardware"},
			{Name: "Finishes & Fixtures", Slug: "finishes-fixtures", Description: "Paint, tiles, plumbing, and lighting"},
		},
		"automotive": {
			{Name: "Auto Parts", Slug: "auto-parts", Description: "Engine parts, brakes, and transmission"},
			{Name: "Accessories", Slug: "auto-accessories", Description: "Interior and exterior accessories"},
			{Name: "Automotive Tools", Slug: "automotive-tools", Description: "Diagnostic tools and garage equipment"},
		},
	}
	for parentSlug, children := range subs {
		var parent models.Category
		db.Where("slug = ?", parentSlug).First(&parent)
		for i := range children {
			children[i].ParentID = &parent.ID
			db.Create(&children[i])
		}
	}
	fmt.Println("Seeded categories and subcategories")
}
