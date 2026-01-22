package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

// Meter defines a usage measurement unit.
type Meter struct {
	ID          snowflake.ID `json:"id" gorm:"primaryKey"`
	OrgID       snowflake.ID `json:"organization_id" gorm:"column:org_id;not null;index:ux_meters_org_code,priority:1"`
	Code        string       `json:"code" gorm:"type:text;not null;index:ux_meters_org_code,priority:2"`
	Name        string       `json:"name" gorm:"type:text;not null"`
	Aggregation string       `json:"aggregation" gorm:"type:text;not null"`
	Unit        string       `json:"unit" gorm:"type:text;not null"`
	Active      bool         `json:"active" gorm:"not null;default:true"`
	CreatedAt   time.Time    `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time    `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TableName sets the database table name.
func (Meter) TableName() string { return "meters" }
