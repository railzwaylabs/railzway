package domain

import (
	"time"

	"github.com/google/uuid"
	featuredomain "github.com/railzwaylabs/railzway/internal/feature/domain"
)

type FeatureAssignment struct {
	ProductID   uuid.UUID
	FeatureID   uuid.UUID
	Code        string
	Name        string
	FeatureType featuredomain.FeatureType
	MeterID     *uuid.UUID
	Active      bool
	CreatedAt   time.Time
}
