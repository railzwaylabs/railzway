// Package domain contains persistence models for rating outputs.
package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
	billingcycledomain "github.com/smallbiznis/railzway/internal/billingcycle/domain"
)

// RatingResult captures the priced usage output for a billing cycle.
type RatingResult struct {
	ID             snowflake.ID  `gorm:"primaryKey"`
	OrgID          snowflake.ID  `gorm:"not null;index"`
	SubscriptionID snowflake.ID  `gorm:"not null;index"`
	BillingCycleID snowflake.ID  `gorm:"not null;index"`
	PriceID        snowflake.ID  `gorm:"not null"`
	FeatureCode    string        `gorm:"type:text"`
	MeterID        *snowflake.ID `gorm:"index"`
	Quantity       float64       `gorm:"not null"`
	UnitPrice      int64         `gorm:"not null"`
	Amount         int64         `gorm:"not null"`
	Currency       string        `gorm:"type:text;not null"`
	PeriodStart    time.Time     `gorm:"not null"`
	PeriodEnd      time.Time     `gorm:"not null"`
	Source         string        `gorm:"type:text;not null"`
	Checksum       string        `gorm:"type:text;not null;uniqueIndex"`
	CreatedAt      time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TableName sets the database table name.
func (RatingResult) TableName() string { return "rating_results" }

type BillingCycleRow struct {
	ID             snowflake.ID
	OrgID          snowflake.ID
	SubscriptionID snowflake.ID
	PeriodStart    time.Time
	PeriodEnd      time.Time
	Status         billingcycledomain.BillingCycleStatus
}

type SubscriptionItemRow struct {
	ID             snowflake.ID
	OrgID          snowflake.ID
	SubscriptionID snowflake.ID
	PriceID        snowflake.ID
	MeterID        *snowflake.ID
}
