// Package domain contains persistence models for feature flags.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type FeatureFlag struct {
	ID        uuid.UUID  `gorm:"primaryKey" json:"id"`
	OrgID     *uuid.UUID `gorm:"index" json:"org_id,omitempty"`
	Key       string     `gorm:"type:text;not null" json:"key"`
	Enabled   bool       `gorm:"not null" json:"enabled"`
	Rollout   int        `gorm:"not null" json:"rollout"`
	CreatedAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (FeatureFlag) TableName() string { return "feature_flags" }

type FeatureFlagAudit struct {
	ID        uuid.UUID  `gorm:"primaryKey" json:"id"`
	OrgID     *uuid.UUID `gorm:"index" json:"org_id,omitempty"`
	Key       string     `gorm:"type:text;not null" json:"key"`
	Enabled   bool       `gorm:"not null" json:"enabled"`
	Rollout   int        `gorm:"not null" json:"rollout"`
	ActorID   string     `gorm:"type:text;not null" json:"actor_id"`
	CreatedAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (FeatureFlagAudit) TableName() string { return "feature_flag_audits" }
