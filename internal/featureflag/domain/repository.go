package domain

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	WithTx(tx *gorm.DB) Repository

	FindFlag(ctx context.Context, orgID *uuid.UUID, key string) (*FeatureFlag, error)
	ListFlags(ctx context.Context, orgID *uuid.UUID) ([]FeatureFlag, error)
	UpsertFlag(ctx context.Context, flag FeatureFlag) error
	CreateAudit(ctx context.Context, audit FeatureFlagAudit) error
}
