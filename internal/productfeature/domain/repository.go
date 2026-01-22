package domain

import (
	"context"
	"time"

	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

type Repository interface {
	ListByProduct(ctx context.Context, db *gorm.DB, orgID, productID snowflake.ID) ([]FeatureAssignment, error)
	ListByProducts(ctx context.Context, db *gorm.DB, orgID snowflake.ID, productIDs []snowflake.ID) ([]FeatureAssignment, error)
	Replace(ctx context.Context, db *gorm.DB, productID snowflake.ID, featureIDs []snowflake.ID, now time.Time) error
}
