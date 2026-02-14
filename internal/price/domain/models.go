package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
	"gorm.io/datatypes"
)

type PricingModel string

var (
	Flat            PricingModel = "FLAT"
	PerUnit         PricingModel = "PER_UNIT"
	TieredVolume    PricingModel = "TIERED_VOLUME"
	TieredGraduated PricingModel = "TIERED_GRADUATED"
)

type BillingUnit string

var (
	API_CALL BillingUnit = "API_CALL"
	GB       BillingUnit = "GB"
	GiB      BillingUnit = "GiB"
	MB       BillingUnit = "MB"
	MiB      BillingUnit = "MiB"
	Second   BillingUnit = "SECOND"
	Minute   BillingUnit = "MINUTE"
	Hour     BillingUnit = "HOUR"
	Seat     BillingUnit = "SEAT"
)

type BillingMode string

var (
	Licensed BillingMode = "LICENSED"
	Metered  BillingMode = "METERED"
)

type BillingInterval string

var (
	Day   BillingInterval = "DAY"
	Week  BillingInterval = "WEEK"
	Month BillingInterval = "MONTH"
	Year  BillingInterval = "YEAR"
)

type AggregateUsage string

var (
	SUM  AggregateUsage = "SUM"
	MAX  AggregateUsage = "MAX"
	LAST AggregateUsage = "LAST"
)

type ProrationBehavior string

var (
	None               ProrationBehavior = "NONE"
	CreateProration    ProrationBehavior = "CREATE_PRORATION"
	Deferred           ProrationBehavior = "DEFERRED"
	InvoiceImmediately ProrationBehavior = "INVOICE_IMMEDIATELY"
)

type TaxBehavior string

var (
	Inclusive TaxBehavior = "INCLUSIVE"
	Exclusive TaxBehavior = "EXCLUSIVE"
	Inline    TaxBehavior = "INLINE"
)

type Price struct {
	ID                   snowflake.ID      `json:"id" gorm:"primaryKey"`
	OrgID                snowflake.ID      `json:"organization_id" gorm:"column:org_id;not null;index"`
	ProductID            snowflake.ID      `json:"product_id" gorm:"column:product_id;not null;index"`
	Code                 string            `json:"code" gorm:"type:text;not null"`
	Name                 string            `json:"name,omitempty" gorm:"type:text"`
	Description          string            `json:"description,omitempty" gorm:"type:text"`
	IdempotencyKey       *string           `json:"-" gorm:"column:idempotency_key"`
	PricingModel         PricingModel      `json:"pricing_model" gorm:"type:text;not null;default:0"`
	BillingMode          BillingMode       `json:"billing_mode" gorm:"type:text;not null;default:0"`
	BillingInterval      BillingInterval   `json:"billing_interval" gorm:"type:text;not null;default:0"`
	BillingIntervalCount int32             `json:"billing_interval_count" gorm:"not null;default:1"`
	AggregateUsage       *AggregateUsage   `json:"aggregate_usage,omitempty" gorm:"type:text"`
	BillingUnit          *BillingUnit      `json:"billing_unit,omitempty" gorm:"type:text"`
	BillingThreshold     *float64          `json:"billing_threshold,omitempty" gorm:"type:numeric"`
	TaxBehavior          TaxBehavior       `json:"tax_behavior" gorm:"type:text;not null;default:0"`
	TaxCode              *string           `json:"tax_code,omitempty" gorm:"type:text"`
	Version              int32             `json:"version" gorm:"not null;default:1"`
	IsDefault            bool              `json:"is_default" gorm:"not null;default:false"`
	Active               bool              `json:"active" gorm:"not null;default:true"`
	RetiredAt            *time.Time        `json:"retired_at,omitempty" gorm:""`
	Metadata             datatypes.JSONMap `json:"metadata" gorm:"type:jsonb"`
	CreatedAt            time.Time         `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt            time.Time         `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (Price) TableName() string { return "prices" }
