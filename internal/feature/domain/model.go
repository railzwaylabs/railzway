package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
	"gorm.io/datatypes"
)

type FeatureType string

const (
	FeatureTypeBoolean FeatureType = "boolean"
	FeatureTypeMetered FeatureType = "metered"
)

type Feature struct {
	ID    snowflake.ID `gorm:"primaryKey"`
	OrgID snowflake.ID `gorm:"column:org_id;not null;index:ux_features_org_code,priority:1"`
	Code  string       `gorm:"type:text;not null;index:ux_features_org_code,priority:2"`

	Name        string            `gorm:"type:text;not null"`
	Description *string           `gorm:"type:text"`
	Type        FeatureType       `gorm:"column:feature_type;type:text;not null"`
	MeterID     *snowflake.ID     `gorm:"column:meter_id"`
	Active      bool              `gorm:"not null;default:true"`
	Metadata    datatypes.JSONMap `gorm:"type:jsonb"`

	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (Feature) TableName() string { return "features" }
