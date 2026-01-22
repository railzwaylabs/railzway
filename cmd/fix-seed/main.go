package main

import (
	"log"
	"os"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/joho/godotenv"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Product struct {
	ID        int64 `gorm:"primaryKey"`
	OrgID     int64
	Name      string
	Code      string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Price struct {
	ID              int64 `gorm:"primaryKey"`
	OrgID           int64
	ProductID       int64
	Code            string
	Name            string
	PricingModel    string
	BillingMode     string
	BillingInterval string
	TaxBehavior     string
	Active          bool
	Metadata        datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type PriceAmount struct {
	ID              int64 `gorm:"primaryKey"`
	OrgID           int64
	PriceID         int64
	Currency        string
	UnitAmountCents int64
	EffectiveFrom   time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=35411231 dbname=postgres port=5433 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to database")

	// Initialize snowflake node
	node, err := snowflake.NewNode(1)
	if err != nil {
		log.Fatalf("Failed to create snowflake node: %v", err)
	}

	// Get org ID from env or use default
	orgID := int64(2002990275537932288)

	now := time.Now()

	// Create Product
	productID := node.Generate().Int64()
	product := Product{
		ID:        productID,
		OrgID:     orgID,
		Name:      "Railzway Cloud Instance",
		Code:      "railzway-instance",
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := db.Table("products").Create(&product).Error; err != nil {
		log.Printf("Failed to create product: %v", err)
	} else {
		log.Printf("Created product: %s (ID: %d)", product.Name, product.ID)
	}

	// Define pricing tiers
	tiers := []struct {
		Name   string
		Code   string
		Amount int64
		Meta   map[string]interface{}
	}{
		{
			Name:   "Evaluation",
			Code:   "evaluation",
			Amount: 0,
			Meta: map[string]interface{}{
				"badge":       "Evaluation",
				"badge_color": "bg-slate-600",
				"type":        "Personal",
				"description": "For personal projects and experiments.",
				"specs": map[string]string{
					"cpu":       "Shared",
					"ram":       "512MB",
					"storage":   "1GB",
					"isolation": "Light",
				},
			},
		},
		{
			Name:   "Hobby",
			Code:   "hobby",
			Amount: 1900,
			Meta: map[string]interface{}{
				"badge":       "Hobby",
				"badge_color": "bg-blue-600",
				"type":        "Professional",
				"description": "Predictable resources for hobby apps.",
				"specs": map[string]string{
					"cpu":       "1 vCPU",
					"ram":       "1GB",
					"storage":   "10GB",
					"isolation": "Standard",
				},
			},
		},
		{
			Name:   "Production",
			Code:   "production",
			Amount: 3900,
			Meta: map[string]interface{}{
				"badge":       "Production",
				"badge_color": "bg-purple-600",
				"type":        "Business",
				"description": "Dedicated performance for serious workloads.",
				"specs": map[string]string{
					"cpu":       "2 vCPU",
					"ram":       "4GB",
					"storage":   "50GB",
					"isolation": "Dedicated",
				},
			},
		},
		{
			Name:   "Performance",
			Code:   "performance",
			Amount: 9900,
			Meta: map[string]interface{}{
				"badge":       "Performance",
				"badge_color": "bg-amber-600",
				"type":        "Enterprise",
				"description": "Maximum power for high-scale applications.",
				"specs": map[string]string{
					"cpu":       "4 vCPU",
					"ram":       "8GB",
					"storage":   "100GB",
					"isolation": "Dedicated",
				},
			},
		},
	}

	for _, tier := range tiers {
		// Create Price
		priceID := node.Generate().Int64()
		price := Price{
			ID:              priceID,
			OrgID:           orgID,
			ProductID:       productID,
			Code:            tier.Code,
			Name:            tier.Name,
			PricingModel:    "FLAT",
			BillingMode:     "LICENSED",
			BillingInterval: "MONTH",
			TaxBehavior:     "EXCLUSIVE",
			Active:          true,
			Metadata:        datatypes.JSONMap(tier.Meta),
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := db.Table("prices").Create(&price).Error; err != nil {
			log.Printf("Failed to create price %s: %v", tier.Name, err)
			continue
		}
		log.Printf("Created price: %s (ID: %d)", tier.Name, priceID)

		// Create PriceAmount
		amountID := node.Generate().Int64()
		priceAmount := PriceAmount{
			ID:              amountID,
			OrgID:           orgID,
			PriceID:         priceID,
			Currency:        "USD",
			UnitAmountCents: tier.Amount,
			EffectiveFrom:   now,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := db.Table("price_amounts").Create(&priceAmount).Error; err != nil {
			log.Printf("Failed to create price amount for %s: %v", tier.Name, err)
			continue
		}
		log.Printf("Created price amount: %s = $%.2f", tier.Name, float64(tier.Amount)/100)
	}

	log.Println("✅ Pricing data created successfully!")
}
