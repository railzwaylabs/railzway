package domain

import (
	"context"

	"github.com/bwmarrin/snowflake"
	"github.com/railzwaylabs/railzway/pkg/db/pagination"
	"gorm.io/gorm"
)

type Repository interface {
	Insert(ctx context.Context, db *gorm.DB, tier *PriceTier) error
	FindByID(ctx context.Context, db *gorm.DB, orgID, id snowflake.ID) (*PriceTier, error)
	FindByIdempotencyKey(ctx context.Context, db *gorm.DB, orgID snowflake.ID, key string) (*PriceTier, error)
	List(ctx context.Context, db *gorm.DB, orgID snowflake.ID, page pagination.Pagination) ([]*PriceTier, error)
}
