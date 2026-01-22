package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

type SubscriptionEntitlement struct {
	ID             snowflake.ID
	OrgID          snowflake.ID
	SubscriptionID snowflake.ID
	ProductID      snowflake.ID
	FeatureCode    string
	FeatureName    string
	FeatureType    string
	MeterID        *snowflake.ID
	EffectiveFrom  time.Time
	EffectiveTo    *time.Time
	CreatedAt      time.Time
}
