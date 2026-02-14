package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, db *gorm.DB, feature *Feature) error
	FindByID(ctx context.Context, db *gorm.DB, orgID, id int64) (*Feature, error)
	FindByIdempotencyKey(ctx context.Context, db *gorm.DB, orgID int64, key string) (*Feature, error)
	List(ctx context.Context, db *gorm.DB, orgID int64, filter ListRequest, page pagination.Pagination) ([]*Feature, error)
	ListByIDs(ctx context.Context, db *gorm.DB, orgID int64, ids []snowflake.ID) ([]Feature, error)
	Update(ctx context.Context, db *gorm.DB, feature *Feature) error
}
