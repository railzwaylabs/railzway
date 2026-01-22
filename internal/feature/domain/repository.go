package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, db *gorm.DB, feature *Feature) error
	FindByID(ctx context.Context, db *gorm.DB, orgID, id int64) (*Feature, error)
	List(ctx context.Context, db *gorm.DB, orgID int64, filter ListRequest) ([]Feature, error)
	ListByIDs(ctx context.Context, db *gorm.DB, orgID int64, ids []snowflake.ID) ([]Feature, error)
	Update(ctx context.Context, db *gorm.DB, feature *Feature) error
}
