package domain

import (
	"time"

	"github.com/bwmarrin/snowflake"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
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
