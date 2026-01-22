package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
	"gorm.io/datatypes"
)

type PriceTier struct {
	ID              snowflake.ID      `json:"id" gorm:"primaryKey"`
	OrgID           snowflake.ID      `json:"organization_id" gorm:"column:org_id;not null;index"`
	PriceID         snowflake.ID      `json:"price_id" gorm:"column:price_id;not null;index"`
	TierMode        int16             `json:"tier_mode" gorm:"type:smallint;not null;default:0"`
	StartQuantity   float64           `json:"start_quantity" gorm:"type:numeric;not null"`
	EndQuantity     *float64          `json:"end_quantity,omitempty" gorm:"type:numeric"`
	UnitAmountCents *int64            `json:"unit_amount_cents,omitempty" gorm:""`
	FlatAmountCents *int64            `json:"flat_amount_cents,omitempty" gorm:""`
	Unit            string            `json:"unit" gorm:"type:text;not null"`
	Metadata        datatypes.JSONMap `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt       time.Time         `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt       time.Time         `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (PriceTier) TableName() string { return "price_tiers" }
