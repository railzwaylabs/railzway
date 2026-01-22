package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
	featuredomain "github.com/smallbiznis/railzway/internal/feature/domain"
)

type FeatureAssignment struct {
	ProductID   snowflake.ID
	FeatureID   snowflake.ID
	Code        string
	Name        string
	FeatureType featuredomain.FeatureType
	MeterID     *snowflake.ID
	Active      bool
	CreatedAt   time.Time
}
