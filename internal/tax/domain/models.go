// Package domain contains tax persistence models.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TaxRate struct {
	ID         uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID      uuid.UUID       `gorm:"not null;index" json:"org_id"`
	Code       string          `gorm:"type:text;not null" json:"code"`
	Name       string          `gorm:"type:text;not null" json:"name"`
	Percentage float64         `gorm:"not null" json:"percentage"`
	Inclusive  bool            `gorm:"not null" json:"inclusive"`
	Active     bool            `gorm:"not null" json:"active"`
	Metadata   json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt  time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (TaxRate) TableName() string { return "tax_rates" }
