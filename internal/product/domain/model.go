package domain

import (
	"time"

	"gorm.io/datatypes"
)

type Product struct {
	ID          int64             `json:"id" gorm:"primaryKey"`
	OrgID       int64             `json:"organization_id" gorm:"column:org_id;not null;index:ux_products_org_code,priority:1"`
	Code        string            `json:"code" gorm:"type:text;not null;index:ux_products_org_code,priority:2"`
	Name        string            `json:"name" gorm:"type:text;not null"`
	Description *string           `json:"description,omitempty" gorm:"type:text"`
	Active      bool              `json:"active" gorm:"not null;default:true"`
	Metadata    datatypes.JSONMap `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt   time.Time         `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time         `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (Product) TableName() string { return "products" }
