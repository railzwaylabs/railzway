package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID             uuid.UUID       `gorm:"primaryKey" json:"id"`
	OrgID          uuid.UUID       `gorm:"column:org_id;not null;index:ux_products_org_code,priority:1" json:"org_id"`
	Code           string          `gorm:"type:text;not null;index:ux_products_org_code,priority:2" json:"code"`
	Name           string          `gorm:"type:text;not null" json:"name"`
	Description    *string         `gorm:"type:text" json:"description,omitempty"`
	Active         bool            `gorm:"not null;default:true" json:"active"`
	IdempotencyKey *string         `gorm:"column:idempotency_key" json:"-"`
	Metadata       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	CreatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (Product) TableName() string { return "products" }
