package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
)

const (
	ResetPeriodNone          = "none"
	ResetPeriodDay           = "day"
	ResetPeriodMonth         = "month"
	ResetPeriodBillingPeriod = "billing_period"
)

type PlanFeature struct {
	ID           uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID        uuid.UUID       `gorm:"not null;index" json:"org_id"`
	PlanID       uuid.UUID       `gorm:"not null;index" json:"plan_id"`
	FeatureID    uuid.UUID       `gorm:"not null;index" json:"feature_id"`
	Enabled      bool            `gorm:"not null;default:true" json:"enabled"`
	LimitNumeric *float64        `gorm:"" json:"limit_numeric,omitempty"`
	LimitUnit    *string         `gorm:"type:text" json:"limit_unit,omitempty"`
	ResetPeriod  string          `gorm:"type:text;not null;default:'none'" json:"reset_period"`
	Metadata     json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt    time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (PlanFeature) TableName() string { return "plan_features" }

type FeatureAssignment struct {
	PlanID       uuid.UUID
	FeatureID    uuid.UUID
	Code         string
	Name         string
	FeatureType  featuredomain.FeatureType
	MeterID      *uuid.UUID
	Active       bool
	Enabled      bool
	LimitNumeric *float64
	LimitUnit    *string
	ResetPeriod  string
	CreatedAt    time.Time
}
